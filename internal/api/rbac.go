package api

import (
	"net/http"
	"strings"

	"github.com/cerberus8484/opensourcebackup/internal/audit"
	"github.com/cerberus8484/opensourcebackup/internal/auth"
	"github.com/cerberus8484/opensourcebackup/internal/security"
)

// RBACMiddleware validates session tokens and injects user+role into context.
//
// When authEnabled=false (no ADMIN_PASSWORD / ADMIN_EMAIL configured) all
// requests pass through as synthetic admin — matches the old dev-mode behaviour.
func RBACMiddleware(sessions *auth.RBACSessionManager, auditStore audit.Store) func(http.Handler) http.Handler {
	return RBACMiddlewareWithAuth(sessions, auditStore, true)
}

// RBACMiddlewareWithAuth is the configurable version used by main.go.
func RBACMiddlewareWithAuth(sessions *auth.RBACSessionManager, _ audit.Store, authEnabled bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p := r.URL.Path

			// Always public — no session needed
			if isPublicPath(p) || strings.HasPrefix(p, "/ui/") || p == "/" {
				next.ServeHTTP(w, r)
				return
			}

			// Agent routes use their own bearer-token auth
			if strings.HasPrefix(p, "/v1/agent/") {
				next.ServeHTTP(w, r)
				return
			}

			// Dev / no-auth mode: inject synthetic admin session
			if !authEnabled {
				ctx := auth.WithSession(r.Context(), &auth.Session{
					Role:      auth.RoleAdmin,
					UserEmail: "dev-admin",
				})
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Extract + validate session
			token := auth.TokenFromRequest(r)
			session, err := sessions.Get(token)
			if err != nil {
				if strings.HasPrefix(p, "/v1/") {
					writeError(w, http.StatusUnauthorized, "authentication required")
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			r = r.WithContext(auth.WithSession(r.Context(), session))
			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole returns a handler wrapper that checks the caller's role.
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

var _ = security.ClientIPHashed // ensure import is used
