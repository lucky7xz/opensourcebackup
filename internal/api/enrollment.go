package api

import (
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/cerberus8484/opensourcebackup/internal/audit"
	"github.com/cerberus8484/opensourcebackup/internal/auth"
	"github.com/cerberus8484/opensourcebackup/internal/security"
)

const enrollmentTokenTTL = 30 * time.Minute

// createEnrollmentToken handles POST /v1/systems/{id}/enrollment-token.
func (h *Handler) createEnrollmentToken(w http.ResponseWriter, r *http.Request) {
	systemID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid system id")
		return
	}

	raw, err := auth.GenerateToken()
	if err != nil {
		h.log.Error("generate enrollment token", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	expiresAt := time.Now().Add(enrollmentTokenTTL)
	if _, err := h.enrollmentTokens.Create(r.Context(), systemID, auth.HashToken(raw), expiresAt); err != nil {
		h.log.Error("store enrollment token", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	_ = h.auditStore.Append(r.Context(), audit.Event(
		audit.ActionEnrollmentTokenCreated, audit.ResourceAgent, systemID.String()).
		By(audit.ActorAdmin).
		IP(security.ClientIPHashed(r)).
		UA(r.UserAgent()).
		Details("ttl=30m").
		Build())

	writeJSON(w, http.StatusCreated, map[string]any{
		"token":      raw,
		"system_id":  systemID,
		"expires_at": expiresAt,
	})
}

// enrollAgent handles POST /v1/agent/enroll.
func (h *Handler) enrollAgent(w http.ResponseWriter, r *http.Request) {
	var body struct {
		EnrollmentToken string `json:"enrollment_token"`
	}
	if err := decode(r, &body); err != nil {
		handleDecodeError(w, err)
		return
	}
	if body.EnrollmentToken == "" {
		writeError(w, http.StatusBadRequest, "enrollment_token is required")
		return
	}

	hash := auth.HashToken(body.EnrollmentToken)
	et, err := h.enrollmentTokens.Consume(r.Context(), hash)
	if err != nil {
		h.log.Warn("enrollment failed", "reason", err.Error())
		_ = h.auditStore.Append(r.Context(), audit.Event(
			audit.ActionAgentEnrolled, audit.ResourceAgent, "").
			By(audit.ActorAgent).
			IP(security.ClientIPHashed(r)).
			Severity(audit.SeverityWarning).
			Details("enrollment failed: invalid or expired token").
			Failed().Build())
		writeError(w, http.StatusUnauthorized, "invalid or expired enrollment token")
		return
	}

	agentToken, err := auth.GenerateToken()
	if err != nil {
		h.log.Error("generate agent token", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if _, err := h.agentTokens.Create(r.Context(), et.SystemID, auth.HashToken(agentToken)); err != nil {
		h.log.Error("store agent token", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	_ = h.auditStore.Append(r.Context(), audit.Event(
		audit.ActionAgentEnrolled, audit.ResourceAgent, et.SystemID.String()).
		By(audit.ActorAgent).
		IP(security.ClientIPHashed(r)).
		Details("agent successfully enrolled").
		Build())

	writeJSON(w, http.StatusCreated, map[string]any{
		"token":     agentToken,
		"system_id": et.SystemID,
	})
}
