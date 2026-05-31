package api

import (
	"net/http"

	"github.com/cerberus8484/opensourcebackup/internal/health"
)

// handleHealthAlerts handles GET /v1/health/alerts.
// Returns current active alerts derived from the health score.
// Stateless — alerts disappear when the underlying condition is resolved.
func (h *Handler) handleHealthAlerts(w http.ResponseWriter, r *http.Request) {
	scoreResult := computeScore(r.Context(), h) // shared with /v1/health/score
	alerts := health.AlertsFromScore(scoreResult)
	summary := health.Summarize(alerts)

	if alerts == nil {
		alerts = []health.Alert{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"alerts":  alerts,
		"summary": summary,
	})
}
