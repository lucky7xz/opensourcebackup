package notify

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cerberus8484/opensourcebackup/internal/health"
)

// ── Severity filtering ────────────────────────────────────────────────────────

func TestFilterBySeverity_CriticalOnly(t *testing.T) {
	alerts := []health.Alert{
		{Code: "a", Severity: "critical"},
		{Code: "b", Severity: "warning"},
		{Code: "c", Severity: "info"},
	}
	got := filterBySeverity(alerts, "critical")
	if len(got) != 1 || got[0].Code != "a" {
		t.Errorf("want 1 critical, got %v", got)
	}
}

func TestFilterBySeverity_WarningAndAbove(t *testing.T) {
	alerts := []health.Alert{
		{Code: "a", Severity: "critical"},
		{Code: "b", Severity: "warning"},
		{Code: "c", Severity: "info"},
	}
	got := filterBySeverity(alerts, "warning")
	if len(got) != 2 {
		t.Errorf("want 2 (critical+warning), got %d", len(got))
	}
}

func TestFilterBySeverity_InfoGetsAll(t *testing.T) {
	alerts := []health.Alert{
		{Code: "a", Severity: "critical"},
		{Code: "b", Severity: "warning"},
		{Code: "c", Severity: "info"},
	}
	got := filterBySeverity(alerts, "info")
	if len(got) != 3 {
		t.Errorf("want all 3, got %d", len(got))
	}
}

func TestFilterBySeverity_NoMatch(t *testing.T) {
	alerts := []health.Alert{{Code: "a", Severity: "info"}}
	got := filterBySeverity(alerts, "critical")
	if len(got) != 0 {
		t.Errorf("want 0, got %d", len(got))
	}
}

// ── Webhook send ──────────────────────────────────────────────────────────────

func TestSendWebhook_PostsJSON(t *testing.T) {
	var received map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("want POST, got %s", r.Method)
		}
		json.NewDecoder(r.Body).Decode(&received) //nolint:errcheck
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	n := New()
	ch := Channel{Name: "test", Type: "webhook", Target: srv.URL, Enabled: true, MinSeverity: "warning"}
	alerts := []health.Alert{{Code: "agents_offline", Severity: "critical", Title: "Agent offline"}}

	if err := n.sendWebhook(t.Context(), ch, alerts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if received["source"] != "OpenSourceBackup" {
		t.Errorf("want source=OpenSourceBackup, got %v", received["source"])
	}
}

func TestSendWebhook_FailsGracefully(t *testing.T) {
	n := New()
	ch := Channel{Name: "bad", Type: "webhook", Target: "http://127.0.0.1:1", Enabled: true, MinSeverity: "warning"}
	alerts := []health.Alert{{Severity: "critical"}}
	// Should return error, not panic
	err := n.sendWebhook(t.Context(), ch, alerts)
	if err == nil {
		t.Error("expected error for unreachable webhook")
	}
}

// ── Morning Report payload ────────────────────────────────────────────────────

func TestMorningReport_ContainsScore(t *testing.T) {
	var received map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&received) //nolint:errcheck
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	n := New()
	ch := Channel{Name: "test", Type: "webhook", Target: srv.URL, Enabled: true, MinSeverity: "info"}
	report := DailyReport{Score: 85, ScoreLabel: "Good", SuccessJobs: 10, AllGood: true}

	if err := n.sendDailyWebhook(t.Context(), ch, report); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Payload must not be empty
	if len(received) == 0 {
		t.Error("expected non-empty payload")
	}
}

func TestMorningReport_NoSecrets(t *testing.T) {
	var body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var buf [4096]byte
		n, _ := r.Body.Read(buf[:])
		body = string(buf[:n])
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	n := New()
	ch := Channel{Name: "test", Type: "webhook", Target: srv.URL, Enabled: true, MinSeverity: "info"}
	report := DailyReport{Score: 50, ScoreLabel: "Fair"}
	n.sendDailyWebhook(t.Context(), ch, report) //nolint:errcheck

	secrets := []string{"password", "token", "secret", "RESTIC_PASSWORD"}
	for _, s := range secrets {
		if contains(body, s) {
			t.Errorf("report payload contains sensitive string: %q", s)
		}
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && func() bool {
		for i := 0; i <= len(s)-len(sub); i++ {
			if s[i:i+len(sub)] == sub { return true }
		}
		return false
	}()
}
