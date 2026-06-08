package api

import (
	"net/http"

	"github.com/google/uuid"
)

// listAgentRestoreTests handles GET /v1/agent/restore-tests.
// Returns the next pending restore test for the authenticated system (claim).
// Returns empty array when no pending test exists.
func (h *Handler) listAgentRestoreTests(w http.ResponseWriter, r *http.Request) {
	systemID, ok := SystemIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authorization required")
		return
	}
	rt, err := h.restoreTests.ClaimNextPending(r.Context(), systemID)
	if err != nil {
		// ErrNotFound = no pending test — return empty array (not an error)
		writeJSON(w, http.StatusOK, []any{})
		return
	}
	writeJSON(w, http.StatusOK, []any{rt})
}

func (h *Handler) completeAgentRestoreTest(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	systemID, ok := SystemIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authorization required")
		return
	}
	rt, err := h.restoreTests.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, httpStatusForError(err), safeErrorMessage(err))
		return
	}
	if rt.SystemID != systemID {
		writeError(w, http.StatusNotFound, "catalog: not found")
		return
	}
	var body struct {
		VerifiedFiles int   `json:"verified_files"`
		VerifiedBytes int64 `json:"verified_bytes"`
	}
	if err := decode(r, &body); err != nil {
		handleDecodeError(w, err)
		return
	}
	rt.Status = "success"
	rt.VerifiedFiles = &body.VerifiedFiles
	rt.VerifiedBytes = &body.VerifiedBytes
	if err := h.restoreTests.Update(r.Context(), rt); err != nil {
		writeError(w, httpStatusForError(err), safeErrorMessage(err))
		return
	}
	writeJSON(w, http.StatusOK, rt)
}

func (h *Handler) failAgentRestoreTest(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	systemID, ok := SystemIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authorization required")
		return
	}
	rt, err := h.restoreTests.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, httpStatusForError(err), safeErrorMessage(err))
		return
	}
	if rt.SystemID != systemID {
		writeError(w, http.StatusNotFound, "catalog: not found")
		return
	}
	var body struct {
		ErrorSummary string `json:"error_summary"`
	}
	if err := decode(r, &body); err != nil {
		handleDecodeError(w, err)
		return
	}
	rt.Status = "failed"
	rt.ErrorSummary = &body.ErrorSummary
	if err := h.restoreTests.Update(r.Context(), rt); err != nil {
		writeError(w, httpStatusForError(err), safeErrorMessage(err))
		return
	}
	writeJSON(w, http.StatusOK, rt)
}
