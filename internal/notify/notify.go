// Package notify sends outbound notifications (webhook, email) when health
// alerts fire. Channels are configured in the notification_channels table.
package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/cerberus8484/opensourcebackup/internal/health"
)

// Channel represents a configured notification target.
type Channel struct {
	ID          string
	Name        string
	Type        string // "webhook" | "email"
	URL         string // webhook URL or SMTP host
	Target      string // webhook URL or email address
	Enabled     bool
	MinSeverity string // "info" | "warning" | "critical"
}

// Notifier sends notifications for active alerts.
type Notifier struct {
	client *http.Client
	log    *slog.Logger
}

// New creates a Notifier. Logging is disabled by default — attach one with
// WithLogger so dispatch failures are recorded in the structured log.
func New() *Notifier {
	return &Notifier{
		client: &http.Client{Timeout: 10 * time.Second},
		log:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

// WithLogger attaches a structured logger and returns the Notifier for chaining.
func (n *Notifier) WithLogger(l *slog.Logger) *Notifier {
	if l != nil {
		n.log = l
	}
	return n
}

// Send dispatches alerts to all matching enabled channels.
func (n *Notifier) Send(ctx context.Context, channels []Channel, alerts []health.Alert) error {
	if len(alerts) == 0 || len(channels) == 0 {
		return nil
	}
	for _, ch := range channels {
		if !ch.Enabled {
			continue
		}
		matching := filterBySeverity(alerts, ch.MinSeverity)
		if len(matching) == 0 {
			continue
		}
		switch ch.Type {
		case "webhook":
			if err := n.sendWebhook(ctx, ch, matching); err != nil {
				// Non-fatal — log but continue with the remaining channels
				n.log.Warn("notify: webhook dispatch failed", "channel", ch.Name, "error", err)
			}
		case "email":
			// Email via SMTP — placeholder for future implementation
			n.log.Info("notify: email channel skipped — SMTP not yet implemented", "channel", ch.Name)
		}
	}
	return nil
}

func (n *Notifier) sendWebhook(ctx context.Context, ch Channel, alerts []health.Alert) error {
	payload := map[string]any{
		"source":    "OpenSourceBackup",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"alerts":    alerts,
		"summary": map[string]int{
			"total":    len(alerts),
			"critical": countBySeverity(alerts, "critical"),
			"warning":  countBySeverity(alerts, "warning"),
			"info":     countBySeverity(alerts, "info"),
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ch.Target, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "OpenSourceBackup/1.0")

	resp, err := n.client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned %d", resp.StatusCode)
	}
	return nil
}

func filterBySeverity(alerts []health.Alert, minSeverity string) []health.Alert {
	rank := map[string]int{"info": 1, "warning": 2, "critical": 3}
	min := rank[minSeverity]
	var out []health.Alert
	for _, a := range alerts {
		if rank[a.Severity] >= min {
			out = append(out, a)
		}
	}
	return out
}

func countBySeverity(alerts []health.Alert, severity string) int {
	n := 0
	for _, a := range alerts {
		if a.Severity == severity {
			n++
		}
	}
	return n
}
