package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/cerberus8484/opensourcebackup/internal/catalog"
)

// ErrBodyTooLarge is returned by decode when the request body exceeds the configured limit.
var ErrBodyTooLarge = errors.New("request body too large")

// Handler holds all store dependencies for the HTTP API.
type Handler struct {
	systems      catalog.SystemStore
	repositories catalog.RepositoryStore
	policies     catalog.PolicyStore
	jobs         catalog.JobStore
	snapshots    catalog.SnapshotStore
	log          *slog.Logger
}

// New creates a Handler wired to the given stores.
func New(
	systems catalog.SystemStore,
	repositories catalog.RepositoryStore,
	policies catalog.PolicyStore,
	jobs catalog.JobStore,
	snapshots catalog.SnapshotStore,
	log *slog.Logger,
) *Handler {
	return &Handler{
		systems:      systems,
		repositories: repositories,
		policies:     policies,
		jobs:         jobs,
		snapshots:    snapshots,
		log:          log,
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
