package api

import (
	"net/http"

	"github.com/cerberus8484/opensourcebackup/internal/audit"
	"github.com/cerberus8484/opensourcebackup/internal/auth"
	"github.com/cerberus8484/opensourcebackup/internal/security"
)

// RegisterRoutes attaches all v1 API routes to mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", h.health)

	// ── Authentication (public — no session required) ──────────────────────
	// Rate-limited separately: 5 attempts per minute per IP.
	authLimiter := security.NewIPRateLimiter(5.0/60, 5)
	authRateMiddleware := security.RateLimit(authLimiter)
	mux.Handle("POST /auth/login",  authRateMiddleware(http.HandlerFunc(h.handleLogin)))
	mux.Handle("POST /auth/logout", http.HandlerFunc(h.handleLogout))
	mux.Handle("GET /auth/status",  http.HandlerFunc(h.handleAuthStatus))

	// Agent binary downloads
	mux.HandleFunc("GET /downloads/agent", h.listDownloads)
	mux.HandleFunc("GET /downloads/agent/{version}/{platform}", h.downloadAgent)

	// Agent install scripts (served from scripts/ directory)
	mux.HandleFunc("GET /scripts/install-agent.sh", h.serveInstallScript)
	mux.HandleFunc("GET /scripts/install-agent-freebsd.sh", h.serveInstallScript)
	mux.HandleFunc("GET /scripts/install-agent.ps1", h.serveInstallScript)
	// Local (all-in-one) installer script
	mux.HandleFunc("GET /scripts/install-local.ps1", h.serveInstallScript)

	// Enrollment (admin operation — protected by network/future admin auth)
	mux.HandleFunc("POST /v1/systems/{id}/enrollment-token", h.createEnrollmentToken)

	// Agent enrollment (unauthenticated — presents one-time token)
	mux.HandleFunc("POST /v1/agent/enroll", h.enrollAgent)

	// Agent-only routes (protected by AgentAuth middleware)
	agentMux := http.NewServeMux()
	agentMux.HandleFunc("PUT /v1/agent/heartbeat", h.handleAgentHeartbeat)
	agentMux.HandleFunc("GET /v1/agent/jobs", h.listAgentJobs)
	// Retention
	agentMux.HandleFunc("GET  /v1/agent/retention/jobs", h.handleListAgentRetentionJobs)
	agentMux.HandleFunc("POST /v1/agent/retention/validate", h.handleRetentionValidate)
	agentMux.HandleFunc("PUT  /v1/agent/retention/jobs/{id}/complete", h.handleCompleteRetentionJob)
	agentMux.HandleFunc("PUT  /v1/agent/retention/jobs/{id}/fail", h.handleFailRetentionJob)
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

	// ── Audit log (GDPR transparency) ─────────────────────────────────────
	mux.HandleFunc("GET /v1/audit", h.handleAuditLog)

	// ── GDPR — Art. 17 (erasure) + Art. 20 (portability) ─────────────────
	mux.HandleFunc("GET /v1/gdpr/systems/{id}/export", h.handleGDPRExport)
	mux.HandleFunc("DELETE /v1/gdpr/systems/{id}/purge", h.handleGDPRPurge)

	// Web UI — served from WEB_UI_DIR (default: web/dist).
	// All unknown paths fall through to index.html for React Router.
	webDir := h.webUIDir()
	if webDir != "" {
		mux.Handle("/ui/", http.StripPrefix("/ui", spaHandler(webDir)))
		mux.Handle("/", http.RedirectHandler("/ui/", http.StatusFound))
	}
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// unused import guards
var _ = auth.HashToken
var _ = audit.NoopStore{}
var _ = security.ClientIP
