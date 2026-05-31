package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/cerberus8484/opensourcebackup/internal/audit"
	"github.com/cerberus8484/opensourcebackup/internal/auth"
	"github.com/cerberus8484/opensourcebackup/internal/security"
)

// WebAuth returns middleware that protects /v1/ and /ui/ routes with
// a session cookie. Agent routes (/v1/agent/) use their own token-based
// auth and are explicitly excluded.
//
// Public routes (no auth required):
//   - POST /auth/login
//   - GET  /health
//   - GET  /downloads/**
//   - GET  /scripts/**
//   - POST /v1/agent/enroll   (uses enrollment token)
//   - /v1/agent/**            (uses agent bearer token — handled by AgentAuth)
func WebAuth(wa *auth.WebAuthenticator, auditStore audit.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p := r.URL.Path

			// Always public
			if isPublicPath(p) {
				next.ServeHTTP(w, r)
				return
			}

			// Agent routes have their own auth — skip web auth
			if strings.HasPrefix(p, "/v1/agent/") {
				next.ServeHTTP(w, r)
				return
			}

			// Validate session
			token := auth.SessionFromRequest(r)
			if !wa.Validate(token) {
				// API call → 401 JSON
				if strings.HasPrefix(p, "/v1/") {
					writeError(w, http.StatusUnauthorized, "authentication required")
					return
				}
				// UI → redirect to login
				http.Redirect(w, r, "/auth/login", http.StatusFound)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// isPublicPath returns true for paths that do not require authentication.
func isPublicPath(p string) bool {
	public := []string{
		"/health",
		"/auth/login",
		"/auth/logout",
		"/downloads/",
		"/scripts/",
		"/v1/agent/enroll",
	}
	for _, prefix := range public {
		if strings.HasPrefix(p, prefix) {
			return true
		}
	}
	return false
}

// handleLogin handles POST /auth/login.
// Accepts JSON { "password": "..." } or form data.
// Rate-limited by the auth-specific limiter (3 req/min per IP).
func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	type loginRequest struct {
		Password string `json:"password"`
	}
	var req loginRequest
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ip := security.ClientIP(r)
	token, err := h.webAuth.Login(req.Password, ip)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			// Audit failed attempt
			_ = h.auditStore.Append(r.Context(), audit.Entry{
				Action:       audit.ActionLoginFail,
				ResourceType: audit.ResourceAuth,
				Actor:        "admin",
				IP:           ip,
				UserAgent:    r.UserAgent(),
				Details:      "invalid password",
				Success:      false,
			})
			writeError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		h.log.Error("login error", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Audit successful login
	_ = h.auditStore.Append(r.Context(), audit.Entry{
		Action:       audit.ActionLogin,
		ResourceType: audit.ResourceAuth,
		Actor:        "admin",
		IP:           ip,
		UserAgent:    r.UserAgent(),
		Success:      true,
	})

	auth.SetCookie(w, r, token)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleLogout handles POST /auth/logout.
func (h *Handler) handleLogout(w http.ResponseWriter, r *http.Request) {
	token := auth.SessionFromRequest(r)
	if token != "" {
		h.webAuth.Logout(token)
		_ = h.auditStore.Append(r.Context(), audit.Entry{
			Action:       audit.ActionLogout,
			ResourceType: audit.ResourceAuth,
			Actor:        "admin",
			IP:           security.ClientIP(r),
			Success:      true,
		})
	}
	auth.ClearCookie(w)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleAuthStatus handles GET /auth/status — returns whether the session is valid.
func (h *Handler) handleAuthStatus(w http.ResponseWriter, r *http.Request) {
	token := auth.SessionFromRequest(r)
	authenticated := h.webAuth != nil && h.webAuth.Validate(token)
	writeJSON(w, http.StatusOK, map[string]bool{"authenticated": authenticated})
}
