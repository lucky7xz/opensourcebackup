package health

import (
	"testing"
	"time"
)

func ptr(t time.Time) *time.Time { return &t }
func ago(d time.Duration) *time.Time {
	t := time.Now().Add(-d)
	return &t
}

func TestScore_PerfectState_Is100(t *testing.T) {
	in := Input{
		TotalSystems:          2,
		OnlineAgents:          2,
		TotalJobs:             10, SuccessJobs: 10,
		LastSuccessAt:         ago(1 * time.Hour),
		TotalSnapshots:        5, VerifiedSnapshots: 5,
		LastRestoreTestAt:     ago(7 * 24 * time.Hour),
		TotalRepos:            2, UnprotectedRepos: 0, UnencryptedRepos: 0,
		PoliciesWithRetention: 2,
		Now:                   time.Now(),
	}
	r := Calculate(in)
	if r.Score != 100 {
		t.Errorf("want 100, got %d; deductions: %v", r.Score, r.Deductions)
	}
	if r.Label != "Excellent" {
		t.Errorf("want Excellent, got %s", r.Label)
	}
}

func TestScore_NoRestoreTests_Minus30(t *testing.T) {
	in := Input{
		TotalSystems: 1, OnlineAgents: 1,
		TotalJobs: 5, SuccessJobs: 5,
		LastSuccessAt: ago(1 * time.Hour),
		TotalSnapshots: 5, VerifiedSnapshots: 0,
		TotalRepos: 1, UnprotectedRepos: 0, UnencryptedRepos: 0,
		PoliciesWithRetention: 1,
	}
	r := Calculate(in)
	if r.Score != 70 {
		t.Errorf("want 70 (100-30), got %d", r.Score)
	}
	assertCode(t, r, "no_restore_tests")
}

func TestScore_PartialRestoreTests_Minus15(t *testing.T) {
	in := Input{
		TotalSystems: 1, OnlineAgents: 1,
		TotalJobs: 5, SuccessJobs: 5,
		LastSuccessAt:     ago(1 * time.Hour),
		TotalSnapshots:    5, VerifiedSnapshots: 3,
		LastRestoreTestAt: ago(1 * 24 * time.Hour),
		TotalRepos: 1, PoliciesWithRetention: 1,
	}
	r := Calculate(in)
	if r.Score != 85 {
		t.Errorf("want 85 (100-15), got %d; deductions: %v", r.Score, r.Deductions)
	}
}

func TestScore_StaleRestoreTest_Minus10(t *testing.T) {
	in := Input{
		TotalSystems: 1, OnlineAgents: 1,
		TotalJobs: 5, SuccessJobs: 5,
		LastSuccessAt:     ago(1 * time.Hour),
		TotalSnapshots:    3, VerifiedSnapshots: 3,
		LastRestoreTestAt: ago(45 * 24 * time.Hour), // 45 days ago → stale
		TotalRepos: 1, PoliciesWithRetention: 1,
	}
	r := Calculate(in)
	if r.Score != 90 {
		t.Errorf("want 90 (100-10), got %d", r.Score)
	}
	assertCode(t, r, "restore_test_stale")
}

func TestScore_BackupStale24h_Minus15(t *testing.T) {
	in := Input{
		TotalSystems:      1, OnlineAgents: 1,
		TotalJobs:         5, SuccessJobs: 5,
		LastSuccessAt:     ago(36 * time.Hour), // 36h ago
		TotalSnapshots:    3, VerifiedSnapshots: 3,
		LastRestoreTestAt: ago(1 * 24 * time.Hour),
		TotalRepos: 1, PoliciesWithRetention: 1,
	}
	r := Calculate(in)
	if r.Score != 85 {
		t.Errorf("want 85 (100-15), got %d", r.Score)
	}
	assertCode(t, r, "backup_stale_24h")
}

func TestScore_FailedJobsLast24h_Minus20(t *testing.T) {
	in := Input{
		TotalSystems: 1, OnlineAgents: 1,
		TotalJobs: 10, SuccessJobs: 9, FailedJobs: 1, FailedLast24h: 1,
		LastSuccessAt:     ago(1 * time.Hour),
		TotalSnapshots:    3, VerifiedSnapshots: 3,
		LastRestoreTestAt: ago(1 * 24 * time.Hour),
		TotalRepos: 1, PoliciesWithRetention: 1,
	}
	r := Calculate(in)
	if r.Score != 80 {
		t.Errorf("want 80 (100-20), got %d", r.Score)
	}
	assertCode(t, r, "failed_jobs_24h")
}

func TestScore_AgentOffline_Minus10(t *testing.T) {
	in := Input{
		TotalSystems: 2, OnlineAgents: 1, OfflineAgents: 1,
		TotalJobs: 5, SuccessJobs: 5,
		LastSuccessAt:     ago(1 * time.Hour),
		TotalSnapshots:    3, VerifiedSnapshots: 3,
		LastRestoreTestAt: ago(1 * 24 * time.Hour),
		TotalRepos: 1, PoliciesWithRetention: 1,
	}
	r := Calculate(in)
	if r.Score != 90 {
		t.Errorf("want 90 (100-10), got %d", r.Score)
	}
	assertCode(t, r, "agents_offline")
}

func TestScore_UnprotectedRepo_Minus10(t *testing.T) {
	in := Input{
		TotalSystems: 1, OnlineAgents: 1,
		TotalJobs: 5, SuccessJobs: 5,
		LastSuccessAt:     ago(1 * time.Hour),
		TotalSnapshots:    3, VerifiedSnapshots: 3,
		LastRestoreTestAt: ago(1 * 24 * time.Hour),
		TotalRepos: 2, UnprotectedRepos: 1,
		PoliciesWithRetention: 1,
	}
	r := Calculate(in)
	if r.Score != 90 {
		t.Errorf("want 90 (100-10), got %d", r.Score)
	}
	assertCode(t, r, "repos_not_immutable")
}

func TestScore_NoRetention_Minus5(t *testing.T) {
	in := Input{
		TotalSystems: 1, OnlineAgents: 1,
		TotalJobs: 5, SuccessJobs: 5,
		LastSuccessAt:     ago(1 * time.Hour),
		TotalSnapshots:    3, VerifiedSnapshots: 3,
		LastRestoreTestAt: ago(1 * 24 * time.Hour),
		TotalRepos: 1,
		PoliciesWithRetention: 0, // no retention configured
	}
	r := Calculate(in)
	if r.Score != 95 {
		t.Errorf("want 95 (100-5), got %d", r.Score)
	}
	assertCode(t, r, "no_retention_policy")
}

func TestScore_Label_Excellent(t *testing.T) {
	if l, _ := classify(95); l != "Excellent" { t.Errorf("want Excellent") }
	if l, _ := classify(90); l != "Excellent" { t.Errorf("want Excellent at 90") }
}
func TestScore_Label_Good(t *testing.T) {
	if l, _ := classify(80); l != "Good" { t.Errorf("want Good") }
}
func TestScore_Label_Fair(t *testing.T) {
	if l, _ := classify(65); l != "Fair" { t.Errorf("want Fair") }
}
func TestScore_Label_AtRisk(t *testing.T) {
	if l, _ := classify(40); l != "At Risk" { t.Errorf("want At Risk") }
}

func TestScore_FloorIsZero(t *testing.T) {
	in := Input{
		TotalSystems: 5, OfflineAgents: 5,
		TotalJobs: 10, FailedJobs: 10, FailedLast24h: 10,
		TotalSnapshots: 5, VerifiedSnapshots: 0,
		TotalRepos: 3, UnprotectedRepos: 3, UnencryptedRepos: 3,
		PoliciesWithRetention: 0,
	}
	r := Calculate(in)
	if r.Score < 0 {
		t.Errorf("score must never be negative, got %d", r.Score)
	}
}

func TestScore_EmptySystem_Is100(t *testing.T) {
	r := Calculate(Input{Now: time.Now()})
	if r.Score != 100 {
		t.Errorf("empty system should score 100, got %d", r.Score)
	}
}

// assertCode checks that a deduction with the given code exists.
func assertCode(t *testing.T, r ScoreResult, code string) {
	t.Helper()
	for _, d := range r.Deductions {
		if d.Code == code {
			return
		}
	}
	t.Errorf("expected deduction code %q not found in %v", code, r.Deductions)
}
