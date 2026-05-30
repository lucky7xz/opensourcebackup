package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/cerberus8484/opensourcebackup/internal/api"
	"github.com/cerberus8484/opensourcebackup/internal/auth"
)

func TestAgentAuth_Returns401_WhenNoToken(t *testing.T) {
	store := newStubAgentTokenStore()
	handler := api.AgentAuth(store)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/v1/agent/jobs", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", rec.Code)
	}
}

func TestAgentAuth_Returns401_WhenTokenInvalid(t *testing.T) {
	store := newStubAgentTokenStore()
	handler := api.AgentAuth(store)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/v1/agent/jobs", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", rec.Code)
	}
}

func TestAgentAuth_Passes_WhenTokenValid(t *testing.T) {
	store := newStubAgentTokenStore()
	systemID := uuid.New()
	rawToken := "valid-test-token"
	store.Create(context.Background(), systemID, auth.HashToken(rawToken)) //nolint:errcheck

	handler := api.AgentAuth(store)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/v1/agent/jobs", nil)
	req.Header.Set("Authorization", "Bearer "+rawToken)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestAgentAuth_InjectsSystemID_IntoContext(t *testing.T) {
	store := newStubAgentTokenStore()
	systemID := uuid.New()
	rawToken := "ctx-test-token"
	store.Create(context.Background(), systemID, auth.HashToken(rawToken)) //nolint:errcheck

	var gotID uuid.UUID
	handler := api.AgentAuth(store)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotID, _ = api.SystemIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/v1/agent/jobs", nil)
	req.Header.Set("Authorization", "Bearer "+rawToken)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if gotID != systemID {
		t.Errorf("context system_id: want %s, got %s", systemID, gotID)
	}
}
