package scheduler

import (
	"fmt"
	"testing"
	"time"

	"github.com/cerberus8484/opensourcebackup/internal/catalog"
)

// makeConfig builds a ScheduleConfig with the given window.
func makeConfig(start, end string) catalog.ScheduleConfig {
	return catalog.ScheduleConfig{WindowStart: start, WindowEnd: end, Timezone: "UTC"}
}

func TestBackupWindow_NoWindow_AlwaysAllowed(t *testing.T) {
	cfg := makeConfig("", "")
	for _, h := range []int{0, 6, 12, 18, 23} {
		if !inBackupWindow(cfg, time.UTC) {
			t.Errorf("hour %d: expected allowed with no window", h)
		}
	}
}

func TestBackupWindow_DaytimeWindow(t *testing.T) {
	cfg := makeConfig("08:00", "18:00")
	cases := []struct {
		hour, min int
		want      bool
	}{
		{8, 0, true},   // exactly start
		{9, 0, true},   // inside
		{17, 59, true}, // just before end
		{18, 0, false}, // end is exclusive
		{19, 0, false}, // after window
		{7, 59, false}, // before window
		{0, 0, false},  // midnight
	}
	for _, tc := range cases {
		now := time.Date(2026, 1, 1, tc.hour, tc.min, 0, 0, time.UTC)
		got := inBackupWindowAt(cfg, time.UTC, now)
		if got != tc.want {
			t.Errorf("%02d:%02d: want %v got %v", tc.hour, tc.min, tc.want, got)
		}
	}
}

func TestBackupWindow_OvernightWindow(t *testing.T) {
	// 22:00–06:00 spans midnight
	cfg := makeConfig("22:00", "06:00")
	cases := []struct {
		hour, min int
		want      bool
	}{
		{22, 0, true},  // exactly start
		{23, 0, true},  // night
		{0, 0, true},   // midnight
		{2, 0, true},   // deep night
		{5, 59, true},  // just before end
		{6, 0, false},  // end exclusive
		{12, 0, false}, // midday — blocked
		{21, 59, false},// just before start
	}
	for _, tc := range cases {
		now := time.Date(2026, 1, 1, tc.hour, tc.min, 0, 0, time.UTC)
		got := inBackupWindowAt(cfg, time.UTC, now)
		if got != tc.want {
			t.Errorf("%02d:%02d: want %v got %v", tc.hour, tc.min, tc.want, got)
		}
	}
}

// inBackupWindowAt is a testable variant that accepts a fixed "now" time.
func inBackupWindowAt(cfg catalog.ScheduleConfig, loc *time.Location, now time.Time) bool {
	if cfg.WindowStart == "" || cfg.WindowEnd == "" {
		return true
	}
	now = now.In(loc)
	nowMin := now.Hour()*60 + now.Minute()

	parseHHMM := func(s string) int {
		var h, m int
		fmt.Sscanf(s, "%d:%d", &h, &m)
		return h*60 + m
	}
	start := parseHHMM(cfg.WindowStart)
	end := parseHHMM(cfg.WindowEnd)

	if start <= end {
		return nowMin >= start && nowMin < end
	}
	return nowMin >= start || nowMin < end
}
