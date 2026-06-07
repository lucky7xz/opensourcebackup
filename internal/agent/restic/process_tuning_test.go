package restic

import (
	"os"
	"os/exec"
	"testing"
)

func TestLowPriorityEnabled_DefaultOnDisableOff(t *testing.T) {
	orig, had := os.LookupEnv("AGENT_LOW_PRIORITY_RESTIC")
	t.Cleanup(func() {
		if had {
			os.Setenv("AGENT_LOW_PRIORITY_RESTIC", orig)
		} else {
			os.Unsetenv("AGENT_LOW_PRIORITY_RESTIC")
		}
	})

	os.Unsetenv("AGENT_LOW_PRIORITY_RESTIC")
	if !lowPriorityEnabled() {
		t.Error("expected low-priority enabled by default (unset)")
	}

	os.Setenv("AGENT_LOW_PRIORITY_RESTIC", "false")
	if lowPriorityEnabled() {
		t.Error("expected disabled when AGENT_LOW_PRIORITY_RESTIC=false")
	}
	if got := ProcessTuningMode(); got != "disabled" {
		t.Errorf("ProcessTuningMode() = %q, want \"disabled\"", got)
	}

	os.Setenv("AGENT_LOW_PRIORITY_RESTIC", "true")
	if got := ProcessTuningMode(); got == "" || got == "disabled" {
		t.Errorf("ProcessTuningMode() = %q, want a non-empty enabled mode", got)
	}
}

// Tuning must never alter the command being run, and must tolerate a nil cmd.
func TestTuneResticProcess_PreservesCommand(t *testing.T) {
	tuneResticProcess(nil) // must not panic

	cmd := exec.Command("restic", "-r", "repo", "snapshots")
	beforePath, beforeArgs := cmd.Path, append([]string(nil), cmd.Args...)

	tuneResticProcess(cmd)

	if cmd.Path != beforePath {
		t.Errorf("cmd.Path changed: %q -> %q", beforePath, cmd.Path)
	}
	if len(cmd.Args) != len(beforeArgs) {
		t.Fatalf("cmd.Args length changed: %v -> %v", beforeArgs, cmd.Args)
	}
	for i := range beforeArgs {
		if cmd.Args[i] != beforeArgs[i] {
			t.Errorf("cmd.Args[%d] changed: %q -> %q", i, beforeArgs[i], cmd.Args[i])
		}
	}
}
