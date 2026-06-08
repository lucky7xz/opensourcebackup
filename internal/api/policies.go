package api

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"

	"github.com/cerberus8484/opensourcebackup/internal/audit"
	"github.com/cerberus8484/opensourcebackup/internal/catalog"
	"github.com/cerberus8484/opensourcebackup/internal/security"
)

func (h *Handler) listPolicies(w http.ResponseWriter, r *http.Request) {
	policies, err := h.policies.List(r.Context())
	if err != nil {
		h.log.Error("list policies", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if policies == nil {
		policies = []catalog.BackupPolicy{}
	}
	writeJSON(w, http.StatusOK, policies)
}

func (h *Handler) createPolicy(w http.ResponseWriter, r *http.Request) {
	var p catalog.BackupPolicy
	if err := decode(r, &p); err != nil {
		handleDecodeError(w, err)
		return
	}
	if p.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if p.Engine == "" {
		writeError(w, http.StatusBadRequest, "engine is required")
		return
	}
	if err := h.policies.Create(r.Context(), &p); err != nil {
		h.log.Error("create policy", "error", err)
		writeError(w, httpStatusForError(err), safeErrorMessage(err))
		return
	}
	_ = h.auditStore.Append(r.Context(), audit.Event(
		audit.ActionPolicyCreated, audit.ResourcePolicy, p.ID.String()).
		By(audit.ActorAdmin).
		IP(security.ClientIPHashed(r)).
		UA(r.UserAgent()).
		Details(fmt.Sprintf("name=%s engine=%s", p.Name, p.Engine)).
		Build())
	h.notifyPoliciesChanged(r.Context())
	writeJSON(w, http.StatusCreated, p)
}

func (h *Handler) getPolicy(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	p, err := h.policies.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, httpStatusForError(err), safeErrorMessage(err))
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *Handler) updatePolicy(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var p catalog.BackupPolicy
	if err := decode(r, &p); err != nil {
		handleDecodeError(w, err)
		return
	}
	p.ID = id
	if err := h.policies.Update(r.Context(), &p); err != nil {
		writeError(w, httpStatusForError(err), safeErrorMessage(err))
		return
	}
	_ = h.auditStore.Append(r.Context(), audit.Event(
		audit.ActionPolicyUpdated, audit.ResourcePolicy, id.String()).
		By(audit.ActorAdmin).
		IP(security.ClientIPHashed(r)).
		UA(r.UserAgent()).
		Details(fmt.Sprintf("name=%s engine=%s", p.Name, p.Engine)).
		Build())
	h.notifyPoliciesChanged(r.Context())
	writeJSON(w, http.StatusOK, p)
}

func (h *Handler) deletePolicy(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.policies.Delete(r.Context(), id); err != nil {
		writeError(w, httpStatusForError(err), safeErrorMessage(err))
		return
	}
	_ = h.auditStore.Append(r.Context(), audit.Event(
		audit.ActionPolicyDeleted, audit.ResourcePolicy, id.String()).
		By(audit.ActorAdmin).
		IP(security.ClientIPHashed(r)).
		Severity(audit.SeverityWarning).
		Build())
	h.notifyPoliciesChanged(r.Context())
	w.WriteHeader(http.StatusNoContent)
}
