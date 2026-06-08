package api

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/cerberus8484/opensourcebackup/internal/health"
	"github.com/cerberus8484/opensourcebackup/internal/notify"
	"github.com/cerberus8484/opensourcebackup/internal/security"
)

// notifyChannel is the API representation of a notification channel.
type notifyChannel struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Target      string    `json:"target"`
	Enabled     bool      `json:"enabled"`
	MinSeverity string    `json:"min_severity"`
	CreatedAt   time.Time `json:"created_at"`
}

// handleListNotifications handles GET /v1/notifications
func (h *Handler) handleListNotifications(w http.ResponseWriter, r *http.Request) {
	channels, err := h.loadChannels(r.Context())
	if err != nil {
		h.log.Warn("list notifications", "error", err)
		writeJSON(w, http.StatusOK, []notifyChannel{})
		return
	}
	writeJSON(w, http.StatusOK, channels)
}

// handleCreateNotification handles POST /v1/notifications
func (h *Handler) handleCreateNotification(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Type        string `json:"type"`
		Target      string `json:"target"`
		Enabled     bool   `json:"enabled"`
		MinSeverity string `json:"min_severity"`
	}
	if err := decode(r, &req); err != nil {
		handleDecodeError(w, err)
		return
	}
	if req.Name == "" || req.Target == "" {
		writeError(w, http.StatusBadRequest, "name and target are required")
		return
	}
	// SSRF protection: validate webhook URL before storing
	if req.Type == "webhook" || req.Type == "" {
		if err := notify.ValidateWebhookURL(req.Target); err != nil {
			writeError(w, http.StatusBadRequest, "invalid webhook URL: "+err.Error())
			return
		}
	}
	if req.MinSeverity == "" {
		req.MinSeverity = "warning"
	}
	if req.Type == "" {
		req.Type = "webhook"
	}

	id := uuid.New()
	_, err := h.db().Exec(r.Context(),
		`INSERT INTO notification_channels (id, name, type, url, target, enabled, min_severity)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		id, req.Name, req.Type, req.Target, req.Target, req.Enabled, req.MinSeverity,
	)
	if err != nil {
		h.log.Error("create notification channel", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": id.String(), "status": "created"})
}

// handleDeleteNotification handles DELETE /v1/notifications/{id}
func (h *Handler) handleDeleteNotification(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	h.db().Exec(r.Context(), `DELETE FROM notification_channels WHERE id=$1`, id) //nolint:errcheck
	w.WriteHeader(http.StatusNoContent)
}

// handleTestNotification handles POST /v1/notifications/{id}/test
func (h *Handler) handleTestNotification(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	channels, _ := h.loadChannels(r.Context())
	var ch *notify.Channel
	for _, c := range channels {
		if c.ID == id.String() {
			ch = &c
			break
		}
	}
	if ch == nil {
		writeError(w, http.StatusNotFound, "channel not found")
		return
	}

	testAlerts := []health.Alert{{
		Code:        "test",
		Severity:    "info",
		Title:       "Test notification from OpenSourceBackup",
		Description: "This is a test notification to verify your webhook is working.",
		Category:    "system",
	}}

	n := notify.New().WithLogger(h.log)
	if err := n.Send(r.Context(), []notify.Channel{*ch}, testAlerts); err != nil {
		writeError(w, http.StatusInternalServerError, "send failed: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})
	h.log.Info("test notification sent", "channel", ch.Name, "ip", security.ClientIPHashed(r))
}

// loadChannels reads all channels from the DB.
func (h *Handler) loadChannels(ctx context.Context) ([]notify.Channel, error) {
	rows, err := h.db().Query(ctx,
		`SELECT id, name, type, target, enabled, min_severity FROM notification_channels ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []notify.Channel
	for rows.Next() {
		var c notify.Channel
		if err := rows.Scan(&c.ID, &c.Name, &c.Type, &c.Target, &c.Enabled, &c.MinSeverity); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// db returns the underlying pgxpool — accessed via the catalog DB pool.
// Handler needs a pool accessor.
func (h *Handler) db() *pgxpool.Pool { return h.dbPool }
