package api

import (
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/cerberus8484/opensourcebackup/internal/catalog"
)

// handleListAgentRetentionJobs handles GET /v1/agent/retention/jobs
// Returns pending retention jobs for the authenticated system.
func (h *Handler) handleListAgentRetentionJobs(w http.ResponseWriter, r *http.Request) {
	systemID, ok := SystemIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authorization required")
		return
	}
	jobs, err := h.jobs.ListPendingRetentionBySystemID(r.Context(), systemID)
	if err != nil {
		h.log.Error("list retention jobs", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if jobs == nil {
		jobs = []catalog.BackupJob{}
	}
	writeJSON(w, http.StatusOK, jobs)
}

// handleRetentionValidate handles POST /v1/agent/retention/validate
// The agent sends the list of snapshot IDs that restic --dry-run would delete.
// The control plane applies the safety rule and returns the approved subset.
//
// Safety rule (hard constraint):
//
//	A snapshot with a successful restore test MUST NOT be deleted if it is the
//	only restore-tested snapshot remaining for that system after the deletion.
//	This guarantees that every system always retains at least one proven
//	recoverable snapshot.
func (h *Handler) handleRetentionValidate(w http.ResponseWriter, r *http.Request) {
	systemID, ok := SystemIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authorization required")
		return
	}

	var req struct {
		PolicyID    string   `json:"policy_id"`
		CandidateIDs []string `json:"candidate_ids"` // snapshot IDs restic would remove
	}
	if err := decode(r, &req); err != nil {
		handleDecodeError(w, err)
		return
	}
	if len(req.CandidateIDs) == 0 {
		writeJSON(w, http.StatusOK, map[string]any{
			"approved_ids": []string{},
			"protected_ids": []string{},
			"reason": "no candidates",
		})
		return
	}

	ctx := r.Context()

	// Load all snapshots for this system to check restore-test coverage.
	// We look at ALL snapshots for the system, not just the policy,
	// because restore tests span system-wide coverage.
	allSnapshots, err := h.snapshots.ListBySystem(ctx, systemID)
	if err != nil {
		h.log.Error("retention validate: list snapshots", "error", err, "system_id", systemID)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Load all restore tests for this system.
	allRTs, err := h.restoreTests.ListBySystemID(ctx, systemID)
	if err != nil {
		h.log.Error("retention validate: list restore tests", "error", err, "system_id", systemID)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Build set: snapshot IDs that have at least one successful restore test.
	verified := make(map[uuid.UUID]bool)
	for _, rt := range allRTs {
		if rt.Status == "success" {
			verified[rt.SnapshotID] = true
		}
	}

	// Build set: snapshot IDs in the candidate list (catalog UUIDs).
	// Note: the agent sends engine snapshot IDs (restic short hash or full ID).
	// We match by EngineSnapshotID.
	candidateEngineIDs := make(map[string]bool, len(req.CandidateIDs))
	for _, id := range req.CandidateIDs {
		candidateEngineIDs[id] = true
	}

	// Map engine snapshot IDs → catalog snapshot (to check restore-test status).
	engineToSnap := make(map[string]catalog.Snapshot, len(allSnapshots))
	for _, sn := range allSnapshots {
		engineToSnap[sn.EngineSnapshotID] = sn
	}

	// Count verified snapshots that would SURVIVE after deletion.
	survivingVerified := 0
	for _, sn := range allSnapshots {
		if verified[sn.ID] && !candidateEngineIDs[sn.EngineSnapshotID] {
			survivingVerified++
		}
	}

	// Apply safety rule:
	// If survivingVerified == 0, protect the most recent verified snapshot
	// from deletion so at least one restore-tested snapshot always remains.
	protected := make(map[string]bool)
	if survivingVerified == 0 {
		// Find the most recent verified snapshot that is in the candidate list.
		var latestVerifiedEngineID string
		var latestTime = int64(0)
		for _, sn := range allSnapshots {
			if verified[sn.ID] && candidateEngineIDs[sn.EngineSnapshotID] {
				t := sn.CreatedAt.Unix()
				if t > latestTime {
					latestTime = t
					latestVerifiedEngineID = sn.EngineSnapshotID
				}
			}
		}
		if latestVerifiedEngineID != "" {
			protected[latestVerifiedEngineID] = true
			h.log.Info("retention: protecting last restore-tested snapshot from deletion",
				"system_id", systemID,
				"policy_id", req.PolicyID,
				"protected_engine_id", latestVerifiedEngineID,
			)
		}
	}

	// Build response.
	approved := make([]string, 0, len(req.CandidateIDs))
	protectedList := make([]string, 0)
	for _, id := range req.CandidateIDs {
		if protected[id] {
			protectedList = append(protectedList, id)
		} else {
			approved = append(approved, id)
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"approved_ids":  approved,
		"protected_ids": protectedList,
		"reason": func() string {
			if len(protectedList) > 0 {
				return "safety rule: last restore-tested snapshot protected from deletion"
			}
			return "all candidates approved"
		}(),
	})
}

// handleCompleteRetentionJob handles PUT /v1/agent/retention/jobs/{id}/complete
func (h *Handler) handleCompleteRetentionJob(w http.ResponseWriter, r *http.Request) {
	jobID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid job id")
		return
	}

	var req struct {
		RemovedEngineIDs []string `json:"removed_engine_ids"` // snapshot IDs actually deleted
	}
	if err := decode(r, &req); err != nil {
		handleDecodeError(w, err)
		return
	}

	ctx := r.Context()

	// Mark job complete.
	job, err := h.jobs.GetByID(ctx, jobID)
	if err != nil {
		writeError(w, httpStatusForError(err), "job not found")
		return
	}
	now := time.Now().UTC()
	job.Status = "success"
	job.FinishedAt = &now
	if err := h.jobs.Update(ctx, job); err != nil {
		writeError(w, httpStatusForError(err), "update job failed")
		return
	}

	// Remove deleted snapshots from catalog.
	// This is best-effort — a failed removal is logged but does not fail the job.
	if len(req.RemovedEngineIDs) > 0 {
		systemID, _ := SystemIDFromContext(ctx)
		allSnaps, err := h.snapshots.ListBySystem(ctx, systemID)
		if err != nil {
			h.log.Warn("retention complete: could not list snapshots to clean catalog",
				"error", err, "job_id", jobID)
		} else {
			removed := make(map[string]bool, len(req.RemovedEngineIDs))
			for _, id := range req.RemovedEngineIDs {
				removed[id] = true
			}
			for _, sn := range allSnaps {
				if removed[sn.EngineSnapshotID] {
					if err := h.snapshots.Delete(ctx, sn.ID); err != nil {
						h.log.Warn("retention: could not remove snapshot from catalog",
							"snapshot_id", sn.ID, "engine_id", sn.EngineSnapshotID, "error", err)
					}
				}
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleFailRetentionJob handles PUT /v1/agent/retention/jobs/{id}/fail
func (h *Handler) handleFailRetentionJob(w http.ResponseWriter, r *http.Request) {
	jobID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid job id")
		return
	}
	var req struct{ Reason string }
	if err := decode(r, &req); err != nil {
		handleDecodeError(w, err)
		return
	}
	job, err := h.jobs.GetByID(r.Context(), jobID)
	if err != nil {
		writeError(w, httpStatusForError(err), "job not found")
		return
	}
	now := time.Now().UTC()
	job.Status = "failed"
	job.FinishedAt = &now
	job.ErrorSummary = &req.Reason
	if err := h.jobs.Update(r.Context(), job); err != nil {
		writeError(w, httpStatusForError(err), "update job failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
