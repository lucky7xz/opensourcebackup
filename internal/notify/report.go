package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/cerberus8484/opensourcebackup/internal/health"
)

// DailyReport is a summary of backup operations for the last 24 hours.
type DailyReport struct {
	GeneratedAt      time.Time
	Period           string
	TotalSystems     int
	SuccessJobs      int
	FailedJobs       int
	TotalBytes       int64
	TotalSnapshots   int
	VerifiedSnapshots int
	Score            int
	ScoreLabel       string
	Alerts           []health.Alert
	AllGood          bool
}

// SendDailyReport sends a morning report to all webhook channels.
func (n *Notifier) SendDailyReport(ctx context.Context, channels []Channel, report DailyReport) error {
	for _, ch := range channels {
		if !ch.Enabled || ch.Type != "webhook" {
			continue
		}
		if err := n.sendDailyWebhook(ctx, ch, report); err != nil {
			fmt.Printf("notify: morning report to %q failed: %v\n", ch.Name, err)
		}
	}
	return nil
}

func (n *Notifier) sendDailyWebhook(ctx context.Context, ch Channel, r DailyReport) error {
	status := "✅ All systems healthy"
	if !r.AllGood {
		status = fmt.Sprintf("⚠ %d failed jobs", r.FailedJobs)
	}

	payload := map[string]any{
		"source":    "OpenSourceBackup Morning Report",
		"timestamp": r.GeneratedAt.UTC().Format(time.RFC3339),
		"text":      fmt.Sprintf("*OpenSourceBackup Daily Report* — %s\n%s", r.GeneratedAt.Format("Mon, 02 Jan 2006"), status),
		"fields": []map[string]any{
			{"title": "Health Score",      "value": fmt.Sprintf("%d/100 — %s", r.Score, r.ScoreLabel), "short": true},
			{"title": "Backup Success",    "value": fmt.Sprintf("%d / %d jobs", r.SuccessJobs, r.SuccessJobs+r.FailedJobs), "short": true},
			{"title": "Snapshots",         "value": fmt.Sprintf("%d total, %d verified", r.TotalSnapshots, r.VerifiedSnapshots), "short": true},
			{"title": "Data Transferred",  "value": fmtBytes(r.TotalBytes), "short": true},
			{"title": "Active Alerts",     "value": fmt.Sprintf("%d", len(r.Alerts)), "short": true},
			{"title": "Protected Systems", "value": fmt.Sprintf("%d", r.TotalSystems), "short": true},
		},
		"color": func() string {
			if r.AllGood { return "good" }
			if r.FailedJobs > 0 { return "danger" }
			return "warning"
		}(),
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
	return nil
}

func fmtBytes(b int64) string {
	if b < 1024 { return fmt.Sprintf("%d B", b) }
	if b < 1024*1024 { return fmt.Sprintf("%.1f KB", float64(b)/1024) }
	if b < 1024*1024*1024 { return fmt.Sprintf("%.1f MB", float64(b)/1024/1024) }
	return fmt.Sprintf("%.2f GB", float64(b)/1024/1024/1024)
}
