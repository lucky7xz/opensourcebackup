package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/cerberus8484/opensourcebackup/internal/catalog"
)

func (h *Handler) listRestoreTests(w http.ResponseWriter, r *http.Request) {
	tests, err := h.restoreTests.List(r.Context())
	if err != nil {
		h.log.Error("list restore tests", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if tests == nil {
		tests = []catalog.RestoreTest{}
	}
	writeJSON(w, http.StatusOK, tests)
}

// createRestoreTest handles POST /v1/restore-tests.
// Accepts snapshot_id, optional target_path, and optional repository_id override.
// If repository_id is omitted, falls back to the snapshot's own repository.
//
//	snapshot → repository_id (default)
//	snapshot.job_id → job.system_id
func (h *Handler) createRestoreTest(w http.ResponseWriter, r *http.Request) {
	var body struct {
		SnapshotID   string  `json:"snapshot_id"`
		TargetPath   *string `json:"target_path"`
		RepositoryID string  `json:"repository_id"` // optional override
	}
	if err := decode(r, &body); err != nil {
		handleDecodeError(w, err)
		return
	}
	snapshotID, err := uuid.Parse(body.SnapshotID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "snapshot_id must be a valid UUID")
		return
	}

	// Resolve chain: snapshot → job → system
	snap, err := h.snapshots.GetByID(r.Context(), snapshotID)
	if err != nil {
		writeError(w, httpStatusForError(err), fmt.Sprintf("snapshot not found: %s", snapshotID))
		return
	}
	job, err := h.jobs.GetByID(r.Context(), snap.JobID)
	if err != nil {
		writeError(w, httpStatusForError(err), "could not resolve job for snapshot")
		return
	}

	// Repository: use override if provided, otherwise fall back to snapshot's repo
	repoID := snap.RepositoryID
	if body.RepositoryID != "" {
		parsed, parseErr := uuid.Parse(body.RepositoryID)
		if parseErr != nil {
			writeError(w, http.StatusBadRequest, "repository_id must be a valid UUID")
			return
		}
		// Verify the repository exists
		if _, repoErr := h.repositories.GetByID(r.Context(), parsed); repoErr != nil {
			writeError(w, httpStatusForError(repoErr), fmt.Sprintf("repository not found: %s", body.RepositoryID))
			return
		}
		repoID = parsed
	}

	rt := &catalog.RestoreTest{
		SnapshotID:   snapshotID,
		SystemID:     job.SystemID,
		RepositoryID: repoID,
		Status:       "pending",
		TargetPath:   body.TargetPath,
	}
	if err := h.restoreTests.Create(r.Context(), rt); err != nil {
		h.log.Error("create restore test", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusCreated, rt)
}

func (h *Handler) getRestoreTest(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	rt, err := h.restoreTests.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, httpStatusForError(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rt)
}

func (h *Handler) startRestoreTest(w http.ResponseWriter, r *http.Request) {
	rt, ok := h.loadRestoreTest(w, r)
	if !ok {
		return
	}
	now := time.Now()
	rt.Status = "running"
	rt.StartedAt = &now
	if err := h.restoreTests.Update(r.Context(), rt); err != nil {
		writeError(w, httpStatusForError(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rt)
}

func (h *Handler) completeRestoreTest(w http.ResponseWriter, r *http.Request) {
	rt, ok := h.loadRestoreTest(w, r)
	if !ok {
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
	now := time.Now()
	rt.Status = "success"
	rt.FinishedAt = &now
	rt.VerifiedFiles = &body.VerifiedFiles
	rt.VerifiedBytes = &body.VerifiedBytes
	if err := h.restoreTests.Update(r.Context(), rt); err != nil {
		writeError(w, httpStatusForError(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rt)
}

func (h *Handler) failRestoreTest(w http.ResponseWriter, r *http.Request) {
	rt, ok := h.loadRestoreTest(w, r)
	if !ok {
		return
	}
	var body struct {
		ErrorSummary string `json:"error_summary"`
	}
	if err := decode(r, &body); err != nil {
		handleDecodeError(w, err)
		return
	}
	now := time.Now()
	rt.Status = "failed"
	rt.FinishedAt = &now
	rt.ErrorSummary = &body.ErrorSummary
	if err := h.restoreTests.Update(r.Context(), rt); err != nil {
		writeError(w, httpStatusForError(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rt)
}

func (h *Handler) deleteRestoreTest(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.restoreTests.Delete(r.Context(), id); err != nil {
		writeError(w, httpStatusForError(err), err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) loadRestoreTest(w http.ResponseWriter, r *http.Request) (*catalog.RestoreTest, bool) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return nil, false
	}
	rt, err := h.restoreTests.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, httpStatusForError(err), err.Error())
		return nil, false
	}
	return rt, true
}
