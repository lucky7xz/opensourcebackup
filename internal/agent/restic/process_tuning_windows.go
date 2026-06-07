//go:build windows

package restic

import (
	"os/exec"
	"syscall"
)

// belowNormalPriorityClass is the Windows BELOW_NORMAL_PRIORITY_CLASS process
// creation flag. Lowering CPU priority keeps a long restic run from starving
// interactive work and other services. Defined locally so this small tuning
// does not pull in golang.org/x/sys/windows as a direct dependency.
const belowNormalPriorityClass = 0x00004000

const lowPriorityModeName = "windows:below_normal"

// applyLowPriority starts restic in the BELOW_NORMAL CPU priority class.
func applyLowPriority(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.CreationFlags |= belowNormalPriorityClass
}
