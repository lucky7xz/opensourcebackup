package api

import (
	"net/http"
	"time"

	"github.com/cerberus8484/opensourcebackup/internal/catalog"
	"github.com/cerberus8484/opensourcebackup/internal/health"
)

// handleHealthScore handles GET /v1/health/score.
// Computes the Backup Health Score from live catalog data.
// This is the single canonical source — Dashboard and Prometheus use the same logic.
func (h *Handler) handleHealthScore(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	now := time.Now()

	// ── Load data ─────────────────────────────────────────────────────────────

	systems, _      := h.systems.List(ctx)
	jobs, _         := h.jobs.List(ctx)
	snapshots, _    := h.snapshots.List(ctx)
	rts, _          := h.restoreTests.List(ctx)
	repos, _        := h.repositories.List(ctx)
	policies, _     := h.policies.List(ctx)

	// ── Agent connectivity ────────────────────────────────────────────────────

	online, idle, offline := 0, 0, 0
	for _, sys := range systems {
		switch agentStatusFromLastSeen(sys.LastSeen, now) {
		case "online":  online++
		case "idle":    idle++
		default:        offline++
		}
	}

	// ── Jobs ─────────────────────────────────────────────────────────────────

	var successJobs, failedJobs, failedLast24h int
	var lastSuccessAt *time.Time
	cutoff24h := now.Add(-24 * time.Hour)

	for _, j := range jobs {
		if j.Type != catalog.JobTypeBackup {
			continue // skip retention jobs
		}
		switch j.Status {
		case "success":
			successJobs++
			if j.FinishedAt != nil && (lastSuccessAt == nil || j.FinishedAt.After(*lastSuccessAt)) {
				lastSuccessAt = j.FinishedAt
			}
		case "failed":
			failedJobs++
			if j.CreatedAt.After(cutoff24h) {
				failedLast24h++
			}
		}
	}

	// ── Snapshots + restore tests ────────────────────────────────────────────

	verified := 0
	var lastRestoreTestAt *time.Time
	snapIDSet := make(map[string]bool, len(snapshots))
	for _, sn := range snapshots {
		snapIDSet[sn.ID.String()] = true
	}
	for _, rt := range rts {
		if rt.Status == "success" && snapIDSet[rt.SnapshotID.String()] {
			verified++
			if rt.FinishedAt != nil && (lastRestoreTestAt == nil || rt.FinishedAt.After(*lastRestoreTestAt)) {
				lastRestoreTestAt = rt.FinishedAt
			}
		}
	}
	// de-duplicate: one verified per snapshot
	verifiedSnaps := countVerifiedSnapshots(snapshots, rts)

	// ── Repositories ─────────────────────────────────────────────────────────

	unprotected, unencrypted := 0, 0
	for _, repo := range repos {
		if !repo.ImmutableMode.IsProtected() {
			unprotected++
		}
		if repo.EncryptionMode == nil || *repo.EncryptionMode == "" {
			unencrypted++
		}
	}

	// ── Retention ────────────────────────────────────────────────────────────

	policiesWithRetention := 0
	for _, p := range policies {
		if p.RetentionPlan.HasRules() {
			policiesWithRetention++
		}
	}

	// ── Calculate ────────────────────────────────────────────────────────────

	_ = verified // used above for dedup
	input := health.Input{
		TotalSystems:          len(systems),
		OnlineAgents:          online,
		IdleAgents:            idle,
		OfflineAgents:         offline,
		TotalJobs:             successJobs + failedJobs,
		SuccessJobs:           successJobs,
		FailedJobs:            failedJobs,
		FailedLast24h:         failedLast24h,
		LastSuccessAt:         lastSuccessAt,
		TotalSnapshots:        len(snapshots),
		VerifiedSnapshots:     verifiedSnaps,
		LastRestoreTestAt:     lastRestoreTestAt,
		TotalRepos:            len(repos),
		UnprotectedRepos:      unprotected,
		UnencryptedRepos:      unencrypted,
		PoliciesWithRetention: policiesWithRetention,
		Now:                   now,
	}

	result := health.Calculate(input)
	writeJSON(w, http.StatusOK, result)
}

// countVerifiedSnapshots counts snapshots with at least one successful restore test.
func countVerifiedSnapshots(snapshots []catalog.Snapshot, rts []catalog.RestoreTest) int {
	verified := make(map[string]bool)
	for _, rt := range rts {
		if rt.Status == "success" {
			verified[rt.SnapshotID.String()] = true
		}
	}
	count := 0
	for _, sn := range snapshots {
		if verified[sn.ID.String()] {
			count++
		}
	}
	return count
}

// agentStatusFromLastSeen classifies an agent as online/idle/offline.
// Must match the Dashboard.tsx and metrics/collector.go thresholds.
func agentStatusFromLastSeen(lastSeen *time.Time, now time.Time) string {
	if lastSeen == nil {
		return "offline"
	}
	age := now.Sub(*lastSeen)
	if age <= health.OnlineThreshold {
		return "online"
	}
	if age <= health.IdleThreshold {
		return "idle"
	}
	return "offline"
}
