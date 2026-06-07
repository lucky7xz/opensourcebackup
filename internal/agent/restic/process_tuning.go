package restic

import (
	"os"
	"os/exec"
)

// B_LOWPRIO_RESTIC — tame agent-spawned restic processes so a long backup is
// less likely to starve the rest of the machine. This lowers OS scheduling
// priority only; it does NOT claim to prevent any crash. Actual I/O intensity
// is additionally bounded by the per-policy bandwidth limit and pause windows.

// lowPriorityEnabled reports whether agent-spawned restic processes should run
// at lowered OS priority. Enabled by default; set AGENT_LOW_PRIORITY_RESTIC=false
// to let restic run at normal priority (e.g. for maximum throughput).
func lowPriorityEnabled() bool {
	return os.Getenv("AGENT_LOW_PRIORITY_RESTIC") != "false"
}

// tuneResticProcess applies OS-specific low-priority settings to a restic
// command before it is started, when enabled. It is a no-op when disabled or on
// platforms without a tuning implementation. Call it after building the *exec.Cmd
// and before Start/Run.
func tuneResticProcess(cmd *exec.Cmd) {
	if cmd != nil && lowPriorityEnabled() {
		applyLowPriority(cmd)
	}
}

// ProcessTuningMode returns a short, honest description of the active restic
// process tuning for logging (e.g. "windows:below_normal", "disabled",
// "none (non-windows)"). It makes no reliability promises.
func ProcessTuningMode() string {
	if !lowPriorityEnabled() {
		return "disabled"
	}
	return lowPriorityModeName
}
