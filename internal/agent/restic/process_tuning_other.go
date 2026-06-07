//go:build !windows

package restic

import "os/exec"

const lowPriorityModeName = "none (non-windows)"

// applyLowPriority is a no-op on non-Windows platforms. Process priority tuning
// is currently only implemented for Windows; on Linux/FreeBSD the agent runs via
// systemd/rc.d and I/O is bounded by the bandwidth limit and pause windows.
func applyLowPriority(_ *exec.Cmd) {}
