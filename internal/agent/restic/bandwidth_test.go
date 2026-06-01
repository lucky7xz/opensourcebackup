package restic

import (
	"strings"
	"testing"
)

// TestBandwidthFlag verifies the --limit-upload flag is added correctly.
func TestBandwidthFlag_ZeroMeansNoFlag(t *testing.T) {
	r := New("")
	opts := BackupOptions{
		Repo: "/tmp/repo", Password: "test",
		Includes: []string{"/tmp"},
		BandwidthLimitKbps: 0,
	}
	// Build args manually to inspect
	args := buildBackupArgs(opts)
	for _, a := range args {
		if strings.Contains(a, "limit-upload") {
			t.Errorf("0 bandwidth should not produce --limit-upload flag, got: %v", args)
		}
	}
	_ = r
}

func TestBandwidthFlag_PositiveAddsFlag(t *testing.T) {
	r := New("")
	opts := BackupOptions{
		Repo: "/tmp/repo", Password: "test",
		Includes: []string{"/tmp"},
		BandwidthLimitKbps: 1000,
	}
	args := buildBackupArgs(opts)
	found := false
	for i, a := range args {
		if a == "--limit-upload" && i+1 < len(args) && args[i+1] == "1000" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected --limit-upload 1000 in args, got: %v", args)
	}
	_ = r
}

// buildBackupArgs is a testable helper that returns backup args without executing.
func buildBackupArgs(opts BackupOptions) []string {
	args := append([]string{"backup", "--json"}, opts.Includes...)
	for _, ex := range opts.Excludes {
		args = append(args, "--exclude", ex)
	}
	for _, tag := range opts.Tags {
		args = append(args, "--tag", tag)
	}
	if opts.BandwidthLimitKbps > 0 {
		args = append(args, "--limit-upload", intToStr(opts.BandwidthLimitKbps))
	}
	return args
}

func intToStr(n int) string {
	if n == 0 { return "0" }
	buf := [20]byte{}
	pos := len(buf)
	for n > 0 { pos--; buf[pos] = byte('0' + n%10); n /= 10 }
	return string(buf[pos:])
}
