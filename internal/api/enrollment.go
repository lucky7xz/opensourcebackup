package api

import (
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/cerberus8484/opensourcebackup/internal/auth"
)

const enrollmentTokenTTL = 30 * time.Minute

// createEnrollmentToken handles POST /v1/systems/{id}/enrollment-token.
// Creates a one-time enrollment token for the given system (admin operation).
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

	// Return the raw token once — never log it
	writeJSON(w, http.StatusCreated, map[string]any{
		"token":      raw,
		"system_id":  systemID,
		"expires_at": expiresAt,
	})
}

// enrollAgent handles POST /v1/agent/enroll.
// Agent presents a one-time enrollment token and receives a long-lived agent token.
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
		// Do not reveal whether the token exists — always 401
		h.log.Warn("enrollment failed", "reason", err.Error())
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

	// Return the raw agent token once — never log it
	writeJSON(w, http.StatusCreated, map[string]any{
		"token":     agentToken,
		"system_id": et.SystemID,
	})
}
