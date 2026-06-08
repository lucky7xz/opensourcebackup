package api

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/cerberus8484/opensourcebackup/internal/audit"
	"github.com/cerberus8484/opensourcebackup/internal/catalog"
	"github.com/cerberus8484/opensourcebackup/internal/security"
)

// cancelJob handles POST /v1/jobs/{id}/cancel — an operator requests a stop of a
// running/pending backup (operational safety, B_JOB_CANCEL). It only flags the
// job; the agent observes the flag and stops restic, then reports "cancelled".
func (h *Handler) cancelJob(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	_ = decode(r, &body) // reason is optional

	if err := h.jobs.RequestCancel(r.Context(), id, body.Reason); err != nil {
		writeError(w, httpStatusForError(err), safeErrorMessage(err))
		return
	}
	// Audit who/when/why — cancelling a backup is an operational safety action.
	_ = h.auditStore.Append(r.Context(), audit.Event(
		audit.ActionBackupCancelled, audit.ResourceJob, id.String()).
		By(audit.ActorAdmin).
		IP(security.ClientIPHashed(r)).
		UA(r.UserAgent()).
		Details(body.Reason).
		Severity(audit.SeverityWarning).
		Build())

	w.WriteHeader(http.StatusAccepted) // 202 — requested; the agent will act
}

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
		writeError(w, httpStatusForError(err), safeErrorMessage(err))
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
		writeError(w, httpStatusForError(err), safeErrorMessage(err))
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
		writeError(w, httpStatusForError(err), safeErrorMessage(err))
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
		writeError(w, httpStatusForError(err), safeErrorMessage(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
