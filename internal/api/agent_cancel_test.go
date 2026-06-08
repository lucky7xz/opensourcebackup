package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/cerberus8484/opensourcebackup/internal/catalog"
)

// The agent learns about an operator's stop request via the cancel-requested endpoint.
func TestCancelStatusAgentJob_ReportsRequest(t *testing.T) {
	mux, jobStore, systemID, token := progressTestEnv(t)

	job := &catalog.BackupJob{SystemID: systemID, PolicyID: uuid.New(), Status: "running"}
	if err := jobStore.Create(context.Background(), job); err != nil {
		t.Fatalf("seed job: %v", err)
	}
	if err := jobStore.RequestCancel(context.Background(), job.ID, "windows update"); err != nil {
		t.Fatalf("request cancel: %v", err)
	}

	req := httptest.NewRequest("GET", "/v1/agent/jobs/"+job.ID.String()+"/cancel-requested", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	var resp struct {
		CancelRequested bool   `json:"cancel_requested"`
		Reason          string `json:"reason"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !resp.CancelRequested {
		t.Error("cancel_requested = false, want true")
	}
	if resp.Reason != "windows update" {
		t.Errorf("reason = %q, want 'windows update'", resp.Reason)
	}
}

// A stopped job is reported as "cancelled" — NOT "failed".
func TestCancelledAgentJob_SetsCancelledStatus(t *testing.T) {
	mux, jobStore, systemID, token := progressTestEnv(t)

	job := &catalog.BackupJob{SystemID: systemID, PolicyID: uuid.New(), Status: "running"}
	if err := jobStore.Create(context.Background(), job); err != nil {
		t.Fatalf("seed job: %v", err)
	}

	req := httptest.NewRequest("PUT", "/v1/agent/jobs/"+job.ID.String()+"/cancelled",
		strings.NewReader(`{"reason":"stopped by operator"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	got, err := jobStore.GetByID(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("get job: %v", err)
	}
	if got.Status != "cancelled" {
		t.Errorf("status = %q, want cancelled", got.Status)
	}
	if got.FinishedAt == nil {
		t.Error("FinishedAt should be set on a cancelled job")
	}
}
