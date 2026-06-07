package api

import (
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/cerberus8484/opensourcebackup/internal/catalog"
)

// listAgentJobs handles GET /v1/agent/jobs — returns pending jobs for the authenticated system only.
func (h *Handler) listAgentJobs(w http.ResponseWriter, r *http.Request) {
	systemID, ok := SystemIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authorization required")
		return
	}
	jobs, err := h.jobs.ListPendingBySystemID(r.Context(), systemID)
	if err != nil {
		h.log.Error("list agent jobs", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if jobs == nil {
		jobs = []catalog.BackupJob{}
	}
	writeJSON(w, http.StatusOK, jobs)
}

// startAgentJob handles PUT /v1/agent/jobs/{id}/start.
func (h *Handler) startAgentJob(w http.ResponseWriter, r *http.Request) {
	job, ok := h.claimJob(w, r)
	if !ok {
		return
	}
	now := time.Now()
	job.Status = "running"
	job.StartedAt = &now
	if err := h.jobs.Update(r.Context(), job); err != nil {
		writeError(w, httpStatusForError(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, job)
}

// completeAgentJob handles PUT /v1/agent/jobs/{id}/complete.
func (h *Handler) completeAgentJob(w http.ResponseWriter, r *http.Request) {
	job, ok := h.claimJob(w, r)
	if !ok {
		return
	}
	var body struct {
		EngineSnapshotID string   `json:"engine_snapshot_id"`
		BytesUploaded    int64    `json:"bytes_uploaded"`
		Paths            []string `json:"paths"`
	}
	if err := decode(r, &body); err != nil {
		handleDecodeError(w, err)
		return
	}

	now := time.Now()
	job.Status = "success"
	job.FinishedAt = &now
	job.BytesUploaded = &body.BytesUploaded
	if err := h.jobs.Update(r.Context(), job); err != nil {
		writeError(w, httpStatusForError(err), err.Error())
		return
	}
	// Pin a successful job to 100% so it never lingers at e.g. 98.7% (best-effort).
	if err := h.jobs.FinalizeProgress(r.Context(), job.ID); err != nil {
		h.log.Error("finalize progress", "error", err)
	}

	// Register snapshot only if policy has a repository
	policy, err := h.policies.GetByID(r.Context(), job.PolicyID)
	if err == nil && policy.RepositoryID != nil && body.EngineSnapshotID != "" {
		snap := &catalog.Snapshot{
			JobID:            job.ID,
			RepositoryID:     *policy.RepositoryID,
			EngineSnapshotID: body.EngineSnapshotID,
			Paths:            body.Paths,
			ChecksumStatus:   "unverified",
		}
		hostname, _ := SystemIDFromContext(r.Context())
		hostnameStr := hostname.String()
		snap.Hostname = &hostnameStr
		if err := h.snapshots.Create(r.Context(), snap); err != nil {
			h.log.Error("create snapshot", "error", err)
		}
	}

	writeJSON(w, http.StatusOK, job)
}

// failAgentJob handles PUT /v1/agent/jobs/{id}/fail.
func (h *Handler) failAgentJob(w http.ResponseWriter, r *http.Request) {
	job, ok := h.claimJob(w, r)
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
	job.Status = "failed"
	job.FinishedAt = &now
	job.ErrorSummary = &body.ErrorSummary
	if err := h.jobs.Update(r.Context(), job); err != nil {
		writeError(w, httpStatusForError(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, job)
}

// progressAgentJob handles PUT /v1/agent/jobs/{id}/progress — live progress updates
// while a backup runs (B_JOB_PROGRESS). Aggregate counters only; no file paths.
func (h *Handler) progressAgentJob(w http.ResponseWriter, r *http.Request) {
	job, ok := h.claimJob(w, r)
	if !ok {
		return
	}
	var p catalog.JobProgress
	if err := decode(r, &p); err != nil {
		handleDecodeError(w, err)
		return
	}
	if err := h.jobs.UpdateProgress(r.Context(), job.ID, p); err != nil {
		writeError(w, httpStatusForError(err), err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// claimJob fetches a job and verifies it belongs to the authenticated system.
func (h *Handler) claimJob(w http.ResponseWriter, r *http.Request) (*catalog.BackupJob, bool) {
	systemID, ok := SystemIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authorization required")
		return nil, false
	}
	jobID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid job id")
		return nil, false
	}
	job, err := h.jobs.GetByID(r.Context(), jobID)
	if err != nil {
		writeError(w, httpStatusForError(err), err.Error())
		return nil, false
	}
	if job.SystemID != systemID {
		// Return 404, not 403 — don't reveal the job exists for other systems
		writeError(w, http.StatusNotFound, "catalog: not found")
		return nil, false
	}
	return job, true
}
