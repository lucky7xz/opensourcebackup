package api_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/cerberus8484/opensourcebackup/internal/api"
	"github.com/cerberus8484/opensourcebackup/internal/auth"
	"github.com/cerberus8484/opensourcebackup/internal/catalog"
)

// progressTestEnv builds a handler whose job + agent-token stores we keep a
// reference to, so we can seed a running job and authenticate as its system.
func progressTestEnv(t *testing.T) (*http.ServeMux, *stubJobStore, uuid.UUID, string) {
	t.Helper()
	jobStore := newStubJobStore()
	tokenStore := newStubAgentTokenStore()
	systemID := uuid.New()
	rawToken := "progress-test-token"
	if _, err := tokenStore.Create(context.Background(), systemID, auth.HashToken(rawToken)); err != nil {
		t.Fatalf("seed token: %v", err)
	}
	h := api.New(
		newStubSystemStore(), newStubRepositoryStore(), newStubPolicyStore(),
		jobStore, newStubSnapshotStore(), newStubRestoreTestStore(),
		newStubEnrollmentTokenStore(), tokenStore, nil,
		slog.New(slog.NewTextHandler(os.Stderr, nil)),
	)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return mux, jobStore, systemID, rawToken
}

func TestProgressAgentJob_PersistsProgress(t *testing.T) {
	mux, jobStore, systemID, token := progressTestEnv(t)

	job := &catalog.BackupJob{SystemID: systemID, PolicyID: uuid.New(), Status: "running"}
	if err := jobStore.Create(context.Background(), job); err != nil {
		t.Fatalf("seed job: %v", err)
	}

	body := `{"phase":"backup","percent":62.4,"bytes_done":123456,"total_bytes":200000,` +
		`"files_done":12,"total_files":30,"throughput_bps":5033}`
	req := httptest.NewRequest("PUT", "/v1/agent/jobs/"+job.ID.String()+"/progress", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("want 204, got %d (%s)", rec.Code, rec.Body.String())
	}
	got, err := jobStore.GetByID(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("get job: %v", err)
	}
	if got.ProgressPercent != 62.4 {
		t.Errorf("ProgressPercent = %v, want 62.4", got.ProgressPercent)
	}
	if got.ProgressBytesDone != 123456 || got.ProgressBytesTotal != 200000 {
		t.Errorf("bytes = %d/%d, want 123456/200000", got.ProgressBytesDone, got.ProgressBytesTotal)
	}
	if got.ProgressThroughputBps != 5033 {
		t.Errorf("throughput = %d, want 5033", got.ProgressThroughputBps)
	}
}

func TestProgressAgentJob_RejectsOtherSystemsJob(t *testing.T) {
	mux, jobStore, _, token := progressTestEnv(t)

	// Job owned by a DIFFERENT system than the authenticated token.
	job := &catalog.BackupJob{SystemID: uuid.New(), PolicyID: uuid.New(), Status: "running"}
	_ = jobStore.Create(context.Background(), job)

	req := httptest.NewRequest("PUT", "/v1/agent/jobs/"+job.ID.String()+"/progress",
		strings.NewReader(`{"phase":"backup","percent":1}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code == http.StatusNoContent {
		t.Errorf("expected rejection for another system's job, got 204")
	}
}
