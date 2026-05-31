package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/cerberus8484/opensourcebackup/internal/catalog"
)

// ActivityBucket is one time bucket in the activity chart.
type ActivityBucket struct {
	Hour        string `json:"hour"`        // "18:00", "19:00", …
	Backups     int    `json:"backups"`
	RestoreTests int   `json:"restore_tests"`
	Failures    int    `json:"failures"`
}

// handleHealthActivity handles GET /v1/health/activity?hours=24
// Returns per-hour job counts for the activity chart.
// Hours defaults to 24, max 48.
func (h *Handler) handleHealthActivity(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	hours := 24
	if v := r.URL.Query().Get("hours"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 48 {
			hours = n
		}
	}

	jobs, err := h.jobs.List(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	rts, err := h.restoreTests.List(ctx)
	if err != nil {
		rts = nil
	}

	now := time.Now().UTC()
	// Round down to the current hour
	nowHour := now.Truncate(time.Hour)
	cutoff := nowHour.Add(-time.Duration(hours-1) * time.Hour)

	// Build bucket map: hour → counts
	type counts struct{ backups, restoreTests, failures int }
	buckets := make(map[time.Time]*counts, hours)
	for i := 0; i < hours; i++ {
		t := cutoff.Add(time.Duration(i) * time.Hour)
		buckets[t] = &counts{}
	}

	hourOf := func(t time.Time) time.Time { return t.UTC().Truncate(time.Hour) }

	for _, j := range jobs {
		h := hourOf(j.CreatedAt)
		b, ok := buckets[h]
		if !ok {
			continue
		}
		if j.Type == catalog.JobTypeRetention {
			continue
		}
		if j.Status == "failed" {
			b.failures++
		} else {
			b.backups++
		}
	}
	for _, rt := range rts {
		h := hourOf(rt.CreatedAt)
		b, ok := buckets[h]
		if !ok {
			continue
		}
		b.restoreTests++
	}

	// Output ordered slice
	result := make([]ActivityBucket, 0, hours)
	for i := 0; i < hours; i++ {
		t := cutoff.Add(time.Duration(i) * time.Hour)
		b := buckets[t]
		// Local time label
		local := t.In(now.Location())
		label := local.Format("15:04")
		result = append(result, ActivityBucket{
			Hour:         label,
			Backups:      b.backups,
			RestoreTests: b.restoreTests,
			Failures:     b.failures,
		})
	}

	writeJSON(w, http.StatusOK, result)
}
