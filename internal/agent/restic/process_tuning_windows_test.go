//go:build windows

package restic

import (
	"os/exec"
	"strconv"
	"strings"
	"testing"
)

// TestApplyLowPriority_ProducesBelowNormal verifies the mechanism B_LOWPRIO_RESTIC
// relies on: the BELOW_NORMAL_PRIORITY_CLASS creation flag must actually result in
// a process whose Windows priority class is "BelowNormal". It spawns a short-lived
// helper, applies the tuning, and inspects the live process priority.
func TestApplyLowPriority_ProducesBelowNormal(t *testing.T) {
	// ~5s helper we can inspect while it runs.
	cmd := exec.Command("ping", "-n", "10", "127.0.0.1")
	applyLowPriority(cmd)

	if err := cmd.Start(); err != nil {
		t.Skipf("could not start helper process: %v", err)
	}
	defer func() { _ = cmd.Process.Kill() }()

	pid := strconv.Itoa(cmd.Process.Pid)
	out, err := exec.Command("powershell", "-NoProfile", "-Command",
		"(Get-Process -Id "+pid+").PriorityClass").CombinedOutput()
	if err != nil {
		t.Skipf("could not query priority class: %v (%s)", err, strings.TrimSpace(string(out)))
	}

	got := strings.TrimSpace(string(out))
	if got != "BelowNormal" {
		t.Errorf("spawned process PriorityClass = %q, want \"BelowNormal\"", got)
	}
}
