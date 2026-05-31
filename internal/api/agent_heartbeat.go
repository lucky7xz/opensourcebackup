package api

import (
	"net/http"
	"time"

	"github.com/cerberus8484/opensourcebackup/internal/audit"
	"github.com/cerberus8484/opensourcebackup/internal/security"
)

// handleAgentHeartbeat handles PUT /v1/agent/heartbeat.
//
// The system ID is taken from the authenticated token context — the agent
// never sends its own ID in the request body (prevents spoofing).
// Response is intentionally minimal: only the server timestamp is returned
// so the agent can detect clock skew if needed.
func (h *Handler) handleAgentHeartbeat(w http.ResponseWriter, r *http.Request) {
	systemID, ok := SystemIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authorization required")
		return
	}

	now := time.Now().UTC()
	if err := h.systems.UpdateLastSeen(r.Context(), systemID, now); err != nil {
		// ErrNotFound means the system was deleted while the agent was running.
		// Return 401 so the agent stops and re-enrollment is required.
		h.log.Warn("heartbeat: system not found — token belongs to deleted system",
			"system_id", systemID,
		)
		writeError(w, http.StatusUnauthorized, "system not found — re-enroll required")
		return
	}

	// Audit heartbeats only at debug level to avoid flooding the audit log.
	// Uncomment for compliance-heavy environments:
	// _ = h.auditStore.Append(r.Context(), audit.Entry{...})
	_ = audit.ActionCreate // keep import used

	// Return server timestamp — agent can use it to detect clock skew.
	writeJSON(w, http.StatusOK, map[string]any{
		"status":     "ok",
		"server_time": now.Format(time.RFC3339),
		"ip_hash":    security.ClientIPHashed(r),
	})
}
