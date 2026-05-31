package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/cerberus8484/opensourcebackup/internal/auth"
	"github.com/cerberus8484/opensourcebackup/internal/catalog"
)

// PolicyChangeNotifier is called after any policy create/update/delete
// so the scheduler can reload its cron entries without a restart.
// Using an interface keeps the API layer decoupled from the scheduler (DIP).
type PolicyChangeNotifier interface {
	PoliciesChanged(ctx context.Context)
}

// ErrBodyTooLarge is returned by decode when the request body exceeds the configured limit.
var ErrBodyTooLarge = errors.New("request body too large")

// Handler holds all store dependencies for the HTTP API.
type Handler struct {
	systems          catalog.SystemStore
	repositories     catalog.RepositoryStore
	policies         catalog.PolicyStore
	jobs             catalog.JobStore
	snapshots        catalog.SnapshotStore
	enrollmentTokens auth.EnrollmentTokenStore
	agentTokens      auth.AgentTokenStore
	policyNotifier   PolicyChangeNotifier // may be nil
	log              *slog.Logger
}

// New creates a Handler wired to the given stores.
func New(
	systems catalog.SystemStore,
	repositories catalog.RepositoryStore,
	policies catalog.PolicyStore,
	jobs catalog.JobStore,
	snapshots catalog.SnapshotStore,
	enrollmentTokens auth.EnrollmentTokenStore,
	agentTokens auth.AgentTokenStore,
	log *slog.Logger,
) *Handler {
	return &Handler{
		systems:          systems,
		repositories:     repositories,
		policies:         policies,
		jobs:             jobs,
		snapshots:        snapshots,
		enrollmentTokens: enrollmentTokens,
		agentTokens:      agentTokens,
		log:              log,
	}
}

// WithPolicyNotifier wires a PolicyChangeNotifier into the handler.
func (h *Handler) WithPolicyNotifier(n PolicyChangeNotifier) *Handler {
	h.policyNotifier = n
	return h
}

// notifyPoliciesChanged calls the notifier if one is registered.
func (h *Handler) notifyPoliciesChanged(ctx context.Context) {
	if h.policyNotifier != nil {
		h.policyNotifier.PoliciesChanged(ctx)
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, "encoding error", http.StatusInternalServerError)
	}
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func decode(r *http.Request, v any) error {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return ErrBodyTooLarge
		}
		return err
	}
	return nil
}

// handleDecodeError writes the appropriate HTTP error for a decode failure.
func handleDecodeError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrBodyTooLarge) {
		writeError(w, http.StatusRequestEntityTooLarge, "request body too large")
		return
	}
	writeError(w, http.StatusBadRequest, "invalid request body")
}

// httpStatusForError maps catalog errors to HTTP status codes.
func httpStatusForError(err error) int {
	switch {
	case errors.Is(err, catalog.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, catalog.ErrConflict):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}
