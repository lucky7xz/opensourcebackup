package api_test

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/cerberus8484/opensourcebackup/internal/api"
	"github.com/cerberus8484/opensourcebackup/internal/catalog"
)

func newTestHandler() *api.Handler {
	return api.New(
		newStubSystemStore(),
		newStubRepositoryStore(),
		newStubPolicyStore(),
		newStubJobStore(),
		newStubSnapshotStore(),
		newStubEnrollmentTokenStore(),
		newStubAgentTokenStore(),
		slog.New(slog.NewTextHandler(os.Stderr, nil)),
	)
}

func newMux(h *api.Handler) *http.ServeMux {
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return mux
}

func TestGetHealth_Returns200(t *testing.T) {
	mux := newMux(newTestHandler())
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/health", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestListSystems_ReturnsEmptyArray_WhenNoSystems(t *testing.T) {
	mux := newMux(newTestHandler())
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/v1/systems", nil))

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
	var systems []catalog.System
	if err := json.NewDecoder(rec.Body).Decode(&systems); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(systems) != 0 {
		t.Errorf("want empty array, got %d items", len(systems))
	}
}

func TestCreateSystem_Returns201_WithID(t *testing.T) {
	mux := newMux(newTestHandler())
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("POST", "/v1/systems", strings.NewReader(`{"Hostname":"web-01","RiskClass":"standard"}`)))

	if rec.Code != http.StatusCreated {
		t.Errorf("want 201, got %d — body: %s", rec.Code, rec.Body.String())
	}
	var s catalog.System
	if err := json.NewDecoder(rec.Body).Decode(&s); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if s.ID == (uuid.UUID{}) {
		t.Error("expected non-zero ID in response")
	}
}

func TestCreateSystem_Returns400_WhenHostnameMissing(t *testing.T) {
	mux := newMux(newTestHandler())
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("POST", "/v1/systems", strings.NewReader(`{}`)))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestGetSystem_Returns200_WhenExists(t *testing.T) {
	h := newTestHandler()
	mux := newMux(h)

	createRec := httptest.NewRecorder()
	mux.ServeHTTP(createRec, httptest.NewRequest("POST", "/v1/systems", strings.NewReader(`{"Hostname":"db-01","RiskClass":"critical"}`)))
	var created catalog.System
	if err := json.NewDecoder(createRec.Body).Decode(&created); err != nil {
		t.Fatalf("decode create: %v", err)
	}

	getRec := httptest.NewRecorder()
	mux.ServeHTTP(getRec, httptest.NewRequest("GET", "/v1/systems/"+created.ID.String(), nil))
	if getRec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", getRec.Code)
	}
}

func TestGetSystem_Returns404_WhenMissing(t *testing.T) {
	mux := newMux(newTestHandler())
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/v1/systems/"+uuid.New().String(), nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", rec.Code)
	}
}

func TestDeleteSystem_Returns204_WhenExists(t *testing.T) {
	h := newTestHandler()
	mux := newMux(h)

	createRec := httptest.NewRecorder()
	mux.ServeHTTP(createRec, httptest.NewRequest("POST", "/v1/systems", strings.NewReader(`{"Hostname":"to-delete","RiskClass":"standard"}`)))
	var created catalog.System
	if err := json.NewDecoder(createRec.Body).Decode(&created); err != nil {
		t.Fatalf("decode create: %v", err)
	}

	delRec := httptest.NewRecorder()
	mux.ServeHTTP(delRec, httptest.NewRequest("DELETE", "/v1/systems/"+created.ID.String(), nil))
	if delRec.Code != http.StatusNoContent {
		t.Errorf("want 204, got %d", delRec.Code)
	}
}

func TestDeleteSystem_Returns404_WhenMissing(t *testing.T) {
	mux := newMux(newTestHandler())
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("DELETE", "/v1/systems/"+uuid.New().String(), nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", rec.Code)
	}
}
