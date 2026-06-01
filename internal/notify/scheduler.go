package notify

import (
	"context"
	"log/slog"
	"time"

	"github.com/cerberus8484/opensourcebackup/internal/catalog"
	"github.com/cerberus8484/opensourcebackup/internal/health"
)

// ReportScheduler sends the morning report every day at a configured time.
type ReportScheduler struct {
	notifier     *Notifier
	channelsFn   func(ctx context.Context) ([]Channel, error)
	scoreFn      func(ctx context.Context) health.ScoreResult
	jobs         catalog.JobStore
	snapshots    catalog.SnapshotStore
	restoreTests catalog.RestoreTestStore
	systems      catalog.SystemStore
	log          *slog.Logger
	sendHour     int // UTC hour to send (default 6 = 06:00 UTC)
}

// NewReportScheduler creates a report scheduler.
func NewReportScheduler(
	notifier *Notifier,
	channelsFn func(ctx context.Context) ([]Channel, error),
	scoreFn func(ctx context.Context) health.ScoreResult,
	jobs catalog.JobStore,
	snaps catalog.SnapshotStore,
	rts catalog.RestoreTestStore,
	systems catalog.SystemStore,
	log *slog.Logger,
) *ReportScheduler {
	return &ReportScheduler{
		notifier: notifier, channelsFn: channelsFn, scoreFn: scoreFn,
		jobs: jobs, snapshots: snaps, restoreTests: rts, systems: systems,
		log: log, sendHour: 6,
	}
}

// Start runs the report loop until ctx is canceled.
func (rs *ReportScheduler) Start(ctx context.Context) {
	for {
		next := rs.nextSendTime()
		rs.log.Info("morning report: next send", "at", next.Format(time.RFC3339))
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Until(next)):
			rs.send(ctx)
		}
	}
}

// nextSendTime returns the next UTC 06:00 after now.
func (rs *ReportScheduler) nextSendTime() time.Time {
	now := time.Now().UTC()
	next := time.Date(now.Year(), now.Month(), now.Day(), rs.sendHour, 0, 0, 0, time.UTC)
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next
}

// send builds and dispatches the daily report.
func (rs *ReportScheduler) send(ctx context.Context) {
	channels, err := rs.channelsFn(ctx)
	if err != nil || len(channels) == 0 {
		return
	}

	cutoff24h := time.Now().Add(-24 * time.Hour)
	jobs, _    := rs.jobs.List(ctx)
	snaps, _   := rs.snapshots.List(ctx)
	rts, _     := rs.restoreTests.List(ctx)
	systems, _ := rs.systems.List(ctx)
	score      := rs.scoreFn(ctx)

	var success, failed int
	var totalBytes int64
	for _, j := range jobs {
		if j.CreatedAt.Before(cutoff24h) { continue }
		if j.Status == "success" {
			success++
			if j.BytesUploaded != nil { totalBytes += *j.BytesUploaded }
		} else if j.Status == "failed" {
			failed++
		}
	}

	verified := 0
	for _, sn := range snaps {
		for _, rt := range rts {
			if rt.SnapshotID == sn.ID && rt.Status == "success" { verified++; break }
		}
	}

	alerts := health.AlertsFromScore(score)

	report := DailyReport{
		GeneratedAt:       time.Now().UTC(),
		Period:            "Last 24 hours",
		TotalSystems:      len(systems),
		SuccessJobs:       success,
		FailedJobs:        failed,
		TotalBytes:        totalBytes,
		TotalSnapshots:    len(snaps),
		VerifiedSnapshots: verified,
		Score:             score.Score,
		ScoreLabel:        score.Label,
		Alerts:            alerts,
		AllGood:           failed == 0 && score.Score >= 75,
	}

	if err := rs.notifier.SendDailyReport(ctx, channels, report); err != nil {
		rs.log.Error("morning report send failed", "error", err)
	} else {
		rs.log.Info("morning report sent", "channels", len(channels), "score", score.Score)
	}
}
