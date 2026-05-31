package api

import (
	"net/http"

	"github.com/cerberus8484/opensourcebackup/internal/auth"
)

// RegisterRoutes attaches all v1 API routes to mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", h.health)

	// Agent binary downloads
	mux.HandleFunc("GET /downloads/agent", h.listDownloads)
	mux.HandleFunc("GET /downloads/agent/{version}/{platform}", h.downloadAgent)

	// Enrollment (admin operation — protected by network/future admin auth)
	mux.HandleFunc("POST /v1/systems/{id}/enrollment-token", h.createEnrollmentToken)

	// Agent enrollment (unauthenticated — presents one-time token)
	mux.HandleFunc("POST /v1/agent/enroll", h.enrollAgent)

	// Agent-only routes (protected by AgentAuth middleware)
	agentMux := http.NewServeMux()
	agentMux.HandleFunc("GET /v1/agent/jobs", h.listAgentJobs)
	agentMux.HandleFunc("PUT /v1/agent/jobs/{id}/start", h.startAgentJob)
	agentMux.HandleFunc("PUT /v1/agent/jobs/{id}/complete", h.completeAgentJob)
	agentMux.HandleFunc("PUT /v1/agent/jobs/{id}/fail", h.failAgentJob)
	// Snapshot read for restore tests (agent needs engine_snapshot_id)
	agentMux.HandleFunc("GET /v1/agent/snapshots/{id}", h.getSnapshot)
	// Restore tests
	agentMux.HandleFunc("GET /v1/agent/restore-tests", h.listAgentRestoreTests)
	agentMux.HandleFunc("PUT /v1/agent/restore-tests/{id}/complete", h.completeAgentRestoreTest)
	agentMux.HandleFunc("PUT /v1/agent/restore-tests/{id}/fail", h.failAgentRestoreTest)
	mux.Handle("/v1/agent/", AgentAuth(h.agentTokens)(agentMux))

	mux.HandleFunc("GET /v1/systems", h.listSystems)
	mux.HandleFunc("POST /v1/systems", h.createSystem)
	mux.HandleFunc("GET /v1/systems/{id}", h.getSystem)
	mux.HandleFunc("PUT /v1/systems/{id}", h.updateSystem)
	mux.HandleFunc("DELETE /v1/systems/{id}", h.deleteSystem)

	mux.HandleFunc("GET /v1/repositories", h.listRepositories)
	mux.HandleFunc("POST /v1/repositories", h.createRepository)
	mux.HandleFunc("GET /v1/repositories/{id}", h.getRepository)
	mux.HandleFunc("PUT /v1/repositories/{id}", h.updateRepository)
	mux.HandleFunc("DELETE /v1/repositories/{id}", h.deleteRepository)

	mux.HandleFunc("GET /v1/policies", h.listPolicies)
	mux.HandleFunc("POST /v1/policies", h.createPolicy)
	mux.HandleFunc("GET /v1/policies/{id}", h.getPolicy)
	mux.HandleFunc("PUT /v1/policies/{id}", h.updatePolicy)
	mux.HandleFunc("DELETE /v1/policies/{id}", h.deletePolicy)

	mux.HandleFunc("GET /v1/jobs", h.listJobs)
	mux.HandleFunc("POST /v1/jobs", h.createJob)
	mux.HandleFunc("GET /v1/jobs/{id}", h.getJob)
	mux.HandleFunc("PUT /v1/jobs/{id}", h.updateJob)
	mux.HandleFunc("DELETE /v1/jobs/{id}", h.deleteJob)

	mux.HandleFunc("GET /v1/snapshots", h.listSnapshots)
	mux.HandleFunc("POST /v1/snapshots", h.createSnapshot)
	mux.HandleFunc("GET /v1/snapshots/{id}", h.getSnapshot)
	mux.HandleFunc("DELETE /v1/snapshots/{id}", h.deleteSnapshot)

	mux.HandleFunc("GET /v1/restore-tests", h.listRestoreTests)
	mux.HandleFunc("POST /v1/restore-tests", h.createRestoreTest)
	mux.HandleFunc("GET /v1/restore-tests/{id}", h.getRestoreTest)
	mux.HandleFunc("PUT /v1/restore-tests/{id}/start", h.startRestoreTest)
	mux.HandleFunc("PUT /v1/restore-tests/{id}/complete", h.completeRestoreTest)
	mux.HandleFunc("PUT /v1/restore-tests/{id}/fail", h.failRestoreTest)
	mux.HandleFunc("DELETE /v1/restore-tests/{id}", h.deleteRestoreTest)
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// unused import guard
var _ = auth.HashToken
