package api

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/cerberus8484/opensourcebackup/internal/catalog"
)

func (h *Handler) listSnapshots(w http.ResponseWriter, r *http.Request) {
	snaps, err := h.snapshots.List(r.Context())
	if err != nil {
		h.log.Error("list snapshots", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if snaps == nil {
		snaps = []catalog.Snapshot{}
	}
	writeJSON(w, http.StatusOK, snaps)
}

func (h *Handler) createSnapshot(w http.ResponseWriter, r *http.Request) {
	var s catalog.Snapshot
	if err := decode(r, &s); err != nil {
		handleDecodeError(w, err)
		return
	}
	if s.JobID == (uuid.UUID{}) {
		writeError(w, http.StatusBadRequest, "job_id is required")
		return
	}
	if s.RepositoryID == (uuid.UUID{}) {
		writeError(w, http.StatusBadRequest, "repository_id is required")
		return
	}
	if s.EngineSnapshotID == "" {
		writeError(w, http.StatusBadRequest, "engine_snapshot_id is required")
		return
	}
	if s.ChecksumStatus == "" {
		s.ChecksumStatus = "unverified"
	}
	if err := h.snapshots.Create(r.Context(), &s); err != nil {
		h.log.Error("create snapshot", "error", err)
		writeError(w, httpStatusForError(err), safeErrorMessage(err))
		return
	}
	writeJSON(w, http.StatusCreated, s)
}

func (h *Handler) getSnapshot(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	s, err := h.snapshots.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, httpStatusForError(err), safeErrorMessage(err))
		return
	}
	writeJSON(w, http.StatusOK, s)
}

func (h *Handler) deleteSnapshot(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.snapshots.Delete(r.Context(), id); err != nil {
		writeError(w, httpStatusForError(err), safeErrorMessage(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
