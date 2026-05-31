package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/cerberus8484/opensourcebackup/internal/audit"
	"github.com/cerberus8484/opensourcebackup/internal/security"
)

func parseUUID(s string) (uuid.UUID, error) { return uuid.Parse(s) }

// handleGDPRExport handles GET /v1/gdpr/systems/{id}/export
// Returns all stored data for a system as JSON (DSGVO Art. 20 — Datenportabilität).
func (h *Handler) handleGDPRExport(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ctx := r.Context()

	uid, err := parseUUID(id)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid system id")
		return
	}

	sys, err := h.systems.GetByID(ctx, uid)
	if err != nil {
		writeError(w, httpStatusForError(err), "system not found")
		return
	}

	jobs, _ := h.jobs.ListBySystemID(ctx, uid)
	snapshots, _ := h.snapshots.ListByJobID(ctx, uid) // best-effort
	restoreTests, _ := h.restoreTests.ListBySystemID(ctx, uid)
	auditEntries, _ := h.auditStore.List(ctx, audit.ResourceSystem, id, 1000)

	export := map[string]any{
		"exported_at":   time.Now().UTC(),
		"gdpr_basis":    "Art. 20 GDPR — Right to data portability",
		"system":        sys,
		"jobs":          jobs,
		"snapshots":     snapshots,
		"restore_tests": restoreTests,
		"audit_log":     auditEntries,
	}

	_ = h.auditStore.Append(ctx, audit.Entry{
		Action:       audit.ActionExport,
		ResourceType: audit.ResourceSystem,
		ResourceID:   id,
		Actor:        "admin",
		IP:           security.ClientIP(r),
		Details:      "GDPR data export requested",
		Success:      true,
	})

	w.Header().Set("Content-Disposition", `attachment; filename="osb-gdpr-export-`+id[:8]+`.json"`)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(export)
}

// handleGDPRPurge handles DELETE /v1/gdpr/systems/{id}/purge
// Hard-deletes ALL data for a system (DSGVO Art. 17 — Recht auf Löschung).
// Writes an audit entry before deleting — the audit entry itself is retained
// for legal accountability (legitimate interest basis).
func (h *Handler) handleGDPRPurge(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ctx := r.Context()

	uid2, err := parseUUID(id)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid system id")
		return
	}

	sys, err := h.systems.GetByID(ctx, uid2)
	if err != nil {
		writeError(w, httpStatusForError(err), "system not found")
		return
	}

	// Write purge audit entry BEFORE deleting — this entry is retained
	// under legitimate interest / legal obligation (Art. 17(3) GDPR).
	_ = h.auditStore.Append(ctx, audit.Entry{
		Action:       audit.ActionPurge,
		ResourceType: audit.ResourceSystem,
		ResourceID:   id,
		Actor:        "admin",
		IP:           security.ClientIP(r),
		Details:      "GDPR Art. 17 erasure request — system: " + sys.Hostname,
		Success:      true,
	})

	// Delete: cascade defined in DB (agent_tokens, enrollment_tokens,
	// jobs, snapshots, restore_tests all have ON DELETE CASCADE).
	if err := h.systems.Delete(ctx, uid2); err != nil {
		writeError(w, httpStatusForError(err), "purge failed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":    "purged",
		"system_id": id,
		"hostname":  sys.Hostname,
		"purged_at": time.Now().UTC(),
		"note":      "All personal data deleted. Audit entry retained per Art. 17(3) GDPR.",
	})
}

// handleAuditLog handles GET /v1/audit
// Returns the audit log for transparency and compliance review.
func (h *Handler) handleAuditLog(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	rt := audit.ResourceType(q.Get("resource_type"))
	rid := q.Get("resource_id")

	entries, err := h.auditStore.List(r.Context(), rt, rid, 200)
	if err != nil {
		h.log.Error("audit list failed", "error", err)
		writeError(w, http.StatusInternalServerError, "could not load audit log")
		return
	}
	if entries == nil {
		entries = []audit.Entry{}
	}
	writeJSON(w, http.StatusOK, entries)
}
