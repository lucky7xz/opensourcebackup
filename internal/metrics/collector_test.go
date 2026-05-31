package metrics

import (
	"testing"
	"time"
)

// ── agentStatus ───────────────────────────────────────────────────────────────

func TestAgentStatus_NilLastSeen_IsOffline(t *testing.T) {
	if got := agentStatus(nil, time.Now()); got != "offline" {
		t.Errorf("want offline, got %s", got)
	}
}

func TestAgentStatus_30Seconds_IsOnline(t *testing.T) {
	now := time.Now()
	ts := now.Add(-30 * time.Second)
	if got := agentStatus(&ts, now); got != "online" {
		t.Errorf("want online, got %s", got)
	}
}

func TestAgentStatus_ExactlyOnlineThreshold_IsOnline(t *testing.T) {
	now := time.Now()
	ts := now.Add(-onlineThreshold)
	if got := agentStatus(&ts, now); got != "online" {
		t.Errorf("want online at threshold, got %s", got)
	}
}

func TestAgentStatus_5Minutes_IsIdle(t *testing.T) {
	now := time.Now()
	ts := now.Add(-5 * time.Minute)
	if got := agentStatus(&ts, now); got != "idle" {
		t.Errorf("want idle, got %s", got)
	}
}

func TestAgentStatus_ExactlyIdleThreshold_IsIdle(t *testing.T) {
	now := time.Now()
	ts := now.Add(-idleThreshold)
	if got := agentStatus(&ts, now); got != "idle" {
		t.Errorf("want idle at threshold, got %s", got)
	}
}

func TestAgentStatus_20Minutes_IsOffline(t *testing.T) {
	now := time.Now()
	ts := now.Add(-20 * time.Minute)
	if got := agentStatus(&ts, now); got != "offline" {
		t.Errorf("want offline, got %s", got)
	}
}

// ── calcRecoveryScore ─────────────────────────────────────────────────────────

func TestRecoveryScore_AllGood_Is100(t *testing.T) {
	score := calcRecoveryScore(5, 5, 0, 0.0)
	if score != 100 {
		t.Errorf("want 100, got %d", score)
	}
}

func TestRecoveryScore_NoSnapshotsAtAll_Is100(t *testing.T) {
	// No snapshots yet = no backups run = no deduction
	score := calcRecoveryScore(0, 0, 0, 0.0)
	if score != 100 {
		t.Errorf("want 100 when no snapshots, got %d", score)
	}
}

func TestRecoveryScore_NoRestoreTests_Minus30(t *testing.T) {
	score := calcRecoveryScore(5, 0, 0, 0.0)
	if score != 70 {
		t.Errorf("want 70 (100-30), got %d", score)
	}
}

func TestRecoveryScore_PartialRestoreTests_Minus15(t *testing.T) {
	score := calcRecoveryScore(5, 3, 0, 0.0)
	if score != 85 {
		t.Errorf("want 85 (100-15), got %d", score)
	}
}

func TestRecoveryScore_FailedJobsLast24h_Minus20(t *testing.T) {
	score := calcRecoveryScore(5, 5, 2, 0.0)
	if score != 80 {
		t.Errorf("want 80 (100-20), got %d", score)
	}
}

func TestRecoveryScore_HighFailureRate_Minus10(t *testing.T) {
	score := calcRecoveryScore(5, 5, 0, 25.0)
	if score != 90 {
		t.Errorf("want 90 (100-10), got %d", score)
	}
}

func TestRecoveryScore_AllBad_Is40(t *testing.T) {
	// Max deductions with current formula: -30 (no tests) -20 (failed 24h) -10 (failure rate) = -60 → 40
	// The floor (0) prevents negative values if the formula is ever extended.
	score := calcRecoveryScore(5, 0, 5, 50.0)
	if score != 40 {
		t.Errorf("want 40 (max deductions = -60), got %d", score)
	}
}

func TestRecoveryScore_Floor_NeverNegative(t *testing.T) {
	// Force a hypothetical negative by passing unrealistic values — floor must hold.
	// This tests the min(0) boundary, not realistic business logic.
	score := calcRecoveryScore(100, 0, 999, 100.0)
	if score < 0 {
		t.Errorf("score must never be negative, got %d", score)
	}
}

func TestRecoveryScore_AtRisk_ComboDeductions(t *testing.T) {
	// -15 partial + -20 failed = 65
	score := calcRecoveryScore(10, 5, 3, 10.0)
	if score != 65 {
		t.Errorf("want 65 (100-15-20), got %d", score)
	}
}

func TestRecoveryScore_Excellent_NoDeductions(t *testing.T) {
	score := calcRecoveryScore(10, 10, 0, 5.0)
	if score != 100 {
		t.Errorf("want 100, got %d", score)
	}
}

// ── countByStatus ─────────────────────────────────────────────────────────────

func TestCountByStatus_Empty(t *testing.T) {
	counts := countByStatus([]string{}, func(s string) string { return s })
	if len(counts) != 0 {
		t.Errorf("want empty map, got %v", counts)
	}
}

func TestCountByStatus_Groups(t *testing.T) {
	items  := []string{"a", "b", "a", "c", "b", "b"}
	counts := countByStatus(items, func(s string) string { return s })

	if counts["a"] != 2 { t.Errorf("a: want 2, got %d", counts["a"]) }
	if counts["b"] != 3 { t.Errorf("b: want 3, got %d", counts["b"]) }
	if counts["c"] != 1 { t.Errorf("c: want 1, got %d", counts["c"]) }
}
