package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/cerberus8484/opensourcebackup/internal/auth"
)

type contextKey string

const systemIDKey contextKey = "authenticated_system_id"

// AgentAuth returns middleware that validates the Bearer token and injects
// the authenticated system_id into the request context.
// Requests without a valid token receive HTTP 401.
func AgentAuth(tokens auth.AgentTokenStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := extractBearer(r)
			if raw == "" {
				writeError(w, http.StatusUnauthorized, "authorization required")
				return
			}
			hash := auth.HashToken(raw)
			systemID, err := tokens.ValidateAndTouch(r.Context(), hash)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "invalid or revoked token")
				return
			}
			next.ServeHTTP(w, r.WithContext(
				context.WithValue(r.Context(), systemIDKey, systemID),
			))
		})
	}
}

// SystemIDFromContext returns the authenticated system_id injected by AgentAuth.
func SystemIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(systemIDKey).(uuid.UUID)
	return id, ok
}

func extractBearer(r *http.Request) string {
	v := r.Header.Get("Authorization")
	if !strings.HasPrefix(v, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(v, "Bearer ")
}
