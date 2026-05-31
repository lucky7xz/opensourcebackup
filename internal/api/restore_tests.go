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
// Accepts only snapshot_id and optional target_path.
// Derives system_id and repository_id from the snapshot chain:
//
//	snapshot → repository_id
//	snapshot.job_id → job.system_id
func (h *Handler) createRestoreTest(w http.ResponseWriter, r *http.Request) {
	var body struct {
		SnapshotID string  `json:"snapshot_id"`
		TargetPath *string `json:"target_path"`
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

	rt := &catalog.RestoreTest{
		SnapshotID:   snapshotID,
		SystemID:     job.SystemID,
		RepositoryID: snap.RepositoryID,
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
