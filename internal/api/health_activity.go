package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/cerberus8484/opensourcebackup/internal/catalog"
)

func ptrVal(p *int64) int64 {
	if p == nil { return 0 }
	return *p
}

// ActivityBucket is one time bucket in the activity chart.
type ActivityBucket struct {
	Hour         string `json:"hour"`          // label: "18:00", "Mon", "01.Jun", …
	Backups      int    `json:"backups"`
	RestoreTests int    `json:"restore_tests"`
	Failures     int    `json:"failures"`
	BytesAdded   int64  `json:"bytes_added"`   // total bytes uploaded in this bucket
}

// handleHealthActivity handles GET /v1/health/activity
//
// Query parameters:
//   hours=N  (default 24, max 48)   → hourly buckets
//   days=N   (7, 30, 90, 365)       → daily buckets
//   weeks=N  (52)                   → weekly buckets
func (h *Handler) handleHealthActivity(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()
	now := time.Now().UTC()

	type mode int
	const (
		modeHours mode = iota
		modeDays
		modeWeeks
	)

	m := modeHours
	n := 24

	if v := q.Get("days"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 && parsed <= 365 {
			m = modeDays
			n = parsed
		}
	} else if v := q.Get("weeks"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 && parsed <= 52 {
			m = modeWeeks
			n = parsed
		}
	} else if v := q.Get("hours"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 && parsed <= 48 {
			n = parsed
		}
	}

	jobs, err := h.jobs.List(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	rts, _ := h.restoreTests.List(ctx)

	type counts struct {
		backups, restoreTests, failures int
		bytes                           int64
	}

	switch m {
	case modeHours:
		nowH := now.Truncate(time.Hour)
		cutoff := nowH.Add(-time.Duration(n-1) * time.Hour)
		buckets := make(map[time.Time]*counts, n)
		for i := 0; i < n; i++ {
			buckets[cutoff.Add(time.Duration(i)*time.Hour)] = &counts{}
		}
		truncH := func(t time.Time) time.Time { return t.UTC().Truncate(time.Hour) }
		for _, j := range jobs {
			b := buckets[truncH(j.CreatedAt)]
			if b == nil || j.Type == catalog.JobTypeRetention { continue }
			if j.Status == "failed" { b.failures++ } else { b.backups++; b.bytes += ptrVal(j.BytesUploaded) }
		}
		for _, rt := range rts {
			b := buckets[truncH(rt.CreatedAt)]
			if b != nil { b.restoreTests++ }
		}
		result := make([]ActivityBucket, n)
		for i := 0; i < n; i++ {
			t := cutoff.Add(time.Duration(i) * time.Hour)
			b := buckets[t]
			result[i] = ActivityBucket{
				Hour: t.In(now.Location()).Format("15:04"),
				Backups: b.backups, RestoreTests: b.restoreTests, Failures: b.failures, BytesAdded: b.bytes,
			}
		}
		writeJSON(w, http.StatusOK, result)

	case modeDays:
		today := now.Truncate(24 * time.Hour)
		cutoff := today.AddDate(0, 0, -(n - 1))
		buckets := make(map[time.Time]*counts, n)
		for i := 0; i < n; i++ {
			buckets[cutoff.AddDate(0, 0, i)] = &counts{}
		}
		truncD := func(t time.Time) time.Time { return t.UTC().Truncate(24 * time.Hour) }
		for _, j := range jobs {
			b := buckets[truncD(j.CreatedAt)]
			if b == nil || j.Type == catalog.JobTypeRetention { continue }
			if j.Status == "failed" { b.failures++ } else { b.backups++; b.bytes += ptrVal(j.BytesUploaded) }
		}
		for _, rt := range rts {
			b := buckets[truncD(rt.CreatedAt)]
			if b != nil { b.restoreTests++ }
		}
		result := make([]ActivityBucket, n)
		for i := 0; i < n; i++ {
			t := cutoff.AddDate(0, 0, i)
			b := buckets[t]
			label := t.In(now.Location()).Format("02.Jan")
			if n <= 7 { label = t.In(now.Location()).Format("Mon") }
			result[i] = ActivityBucket{
				Hour: label,
				Backups: b.backups, RestoreTests: b.restoreTests, Failures: b.failures, BytesAdded: b.bytes,
			}
		}
		writeJSON(w, http.StatusOK, result)

	case modeWeeks:
		// Round to start of current week (Monday)
		wd := int(now.Weekday())
		if wd == 0 { wd = 7 }
		thisWeek := now.Truncate(24*time.Hour).AddDate(0, 0, -(wd-1))
		cutoff := thisWeek.AddDate(0, 0, -(n-1)*7)
		buckets := make(map[time.Time]*counts, n)
		for i := 0; i < n; i++ {
			buckets[cutoff.AddDate(0, 0, i*7)] = &counts{}
		}
		weekOf := func(t time.Time) time.Time {
			d := t.UTC().Truncate(24 * time.Hour)
			w := int(d.Weekday()); if w == 0 { w = 7 }
			return d.AddDate(0, 0, -(w-1))
		}
		for _, j := range jobs {
			b := buckets[weekOf(j.CreatedAt)]
			if b == nil || j.Type == catalog.JobTypeRetention { continue }
			if j.Status == "failed" { b.failures++ } else { b.backups++; b.bytes += ptrVal(j.BytesUploaded) }
		}
		for _, rt := range rts {
			b := buckets[weekOf(rt.CreatedAt)]
			if b != nil { b.restoreTests++ }
		}
		result := make([]ActivityBucket, n)
		for i := 0; i < n; i++ {
			t := cutoff.AddDate(0, 0, i*7)
			b := buckets[t]
			result[i] = ActivityBucket{
				Hour: t.In(now.Location()).Format("02.Jan"),
				Backups: b.backups, RestoreTests: b.restoreTests, Failures: b.failures, BytesAdded: b.bytes,
			}
		}
		writeJSON(w, http.StatusOK, result)
	}
}
