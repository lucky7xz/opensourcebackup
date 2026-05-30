package api

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/cerberus8484/opensourcebackup/internal/catalog"
)

func (h *Handler) listJobs(w http.ResponseWriter, r *http.Request) {
	systemIDStr := r.URL.Query().Get("system_id")
	status := r.URL.Query().Get("status")

	if systemIDStr != "" && status == "pending" {
		id, err := uuid.Parse(systemIDStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid system_id")
			return
		}
		jobs, err := h.jobs.ListPendingBySystemID(r.Context(), id)
		if err != nil {
			h.log.Error("list pending jobs", "error", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		if jobs == nil {
			jobs = []catalog.BackupJob{}
		}
		writeJSON(w, http.StatusOK, jobs)
		return
	}

	jobs, err := h.jobs.List(r.Context())
	if err != nil {
		h.log.Error("list jobs", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if jobs == nil {
		jobs = []catalog.BackupJob{}
	}
	writeJSON(w, http.StatusOK, jobs)
}

func (h *Handler) createJob(w http.ResponseWriter, r *http.Request) {
	var j catalog.BackupJob
	if err := decode(r, &j); err != nil {
		handleDecodeError(w, err)
		return
	}
	if j.SystemID == (uuid.UUID{}) {
		writeError(w, http.StatusBadRequest, "system_id is required")
		return
	}
	if j.PolicyID == (uuid.UUID{}) {
		writeError(w, http.StatusBadRequest, "policy_id is required")
		return
	}
	if j.Status == "" {
		j.Status = "running"
	}
	if err := h.jobs.Create(r.Context(), &j); err != nil {
		h.log.Error("create job", "error", err)
		writeError(w, httpStatusForError(err), err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, j)
}

func (h *Handler) getJob(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	j, err := h.jobs.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, httpStatusForError(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, j)
}

func (h *Handler) updateJob(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var j catalog.BackupJob
	if err := decode(r, &j); err != nil {
		handleDecodeError(w, err)
		return
	}
	j.ID = id
	if err := h.jobs.Update(r.Context(), &j); err != nil {
		writeError(w, httpStatusForError(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, j)
}

func (h *Handler) deleteJob(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.jobs.Delete(r.Context(), id); err != nil {
		writeError(w, httpStatusForError(err), err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
