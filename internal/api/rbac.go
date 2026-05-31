package api

import (
	"net/http"
	"strings"

	"github.com/cerberus8484/opensourcebackup/internal/audit"
	"github.com/cerberus8484/opensourcebackup/internal/auth"
	"github.com/cerberus8484/opensourcebackup/internal/security"
)

// RBACMiddleware replaces the old WebAuth middleware.
// It validates the session token, loads user+role into context,
// and enforces authentication on protected routes.
//
// Public routes (no auth):
//   /health, /auth/*, /downloads/**, /scripts/**, /v1/agent/**, /metrics
func RBACMiddleware(sessions *auth.RBACSessionManager, auditStore audit.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p := r.URL.Path

			// Always public — no session needed
			if isPublicPath(p) {
				next.ServeHTTP(w, r)
				return
			}

			// Agent routes have their own bearer-token auth
			if strings.HasPrefix(p, "/v1/agent/") {
				next.ServeHTTP(w, r)
				return
			}

			// The web UI itself is always accessible — auth happens client-side.
			// Only /v1/ API calls are blocked without a session.
			if strings.HasPrefix(p, "/ui/") || p == "/" {
				next.ServeHTTP(w, r)
				return
			}

			// Extract + validate session
			token := auth.TokenFromRequest(r)
			session, err := sessions.Get(token)
			if err != nil {
				// API call without session → 401
				if strings.HasPrefix(p, "/v1/") {
					writeError(w, http.StatusUnauthorized, "authentication required")
					return
				}
				// Unknown path without session — pass through (let handler decide)
				next.ServeHTTP(w, r)
				return
			}

			// Inject session into context for handlers
			r = r.WithContext(auth.WithSession(r.Context(), session))
			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole returns a handler wrapper that checks the caller's role.
// Call as: RequireRole(auth.RoleAdmin)(h.deleteRepository)
func RequireRole(minimum auth.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := auth.RoleFromContext(r.Context())
			if !role.AtLeast(minimum) {
				writeError(w, http.StatusForbidden,
					"insufficient permissions — role "+string(minimum)+" required")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// requireRoleFn is a convenience wrapper for handler functions.
func requireRoleFn(minimum auth.Role, fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		role := auth.RoleFromContext(r.Context())
		if !role.AtLeast(minimum) {
			writeError(w, http.StatusForbidden,
				"insufficient permissions — role "+string(minimum)+" required")
			return
		}
		fn(w, r)
	}
}

// currentUserMiddleware adds the requesting user's info to the audit trail helper.
// Call security.ClientIPHashed to get the hashed IP; role comes from context.
var _ = security.ClientIPHashed // ensure import is used
