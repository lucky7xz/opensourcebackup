package restic

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// BackupOptions configures a restic backup run.
type BackupOptions struct {
	Repo     string
	Password string
	Includes []string
	Excludes []string
	Tags     []string
}

// BackupResult holds the parsed output of a successful restic backup.
type BackupResult struct {
	SnapshotID   string
	BytesAdded   int64
	FilesNew     int
	FilesChanged int
}

// Runner executes restic commands.
type Runner struct {
	bin string // path to the restic binary
}

// New creates a Runner using the given restic binary path.
// Pass "restic" to rely on PATH.
func New(bin string) *Runner {
	if bin == "" {
		bin = "restic"
	}
	return &Runner{bin: bin}
}

// Backup initializes the repository if needed, then runs a backup.
func (r *Runner) Backup(ctx context.Context, opts BackupOptions) (*BackupResult, error) {
	if err := r.initRepo(ctx, opts); err != nil {
		return nil, err
	}
	return r.runBackup(ctx, opts)
}

func (r *Runner) initRepo(ctx context.Context, opts BackupOptions) error {
	cmd := r.cmd(ctx, opts, "init")
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Ignore: repo already initialized
		if strings.Contains(string(out), "already") ||
			strings.Contains(string(out), "config file already exists") {
			return nil
		}
		return fmt.Errorf("restic init: %w — %s", err, bytes.TrimSpace(out))
	}
	return nil
}

func (r *Runner) runBackup(ctx context.Context, opts BackupOptions) (*BackupResult, error) {
	args := append([]string{"backup", "--json"}, opts.Includes...)
	for _, ex := range opts.Excludes {
		args = append(args, "--exclude", ex)
	}
	for _, tag := range opts.Tags {
		args = append(args, "--tag", tag)
	}

	cmd := r.cmd(ctx, opts, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("restic backup pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("restic backup start: %w", err)
	}

	var result *BackupResult
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Bytes()
		var msg struct {
			MessageType  string `json:"message_type"`
			SnapshotID   string `json:"snapshot_id"`
			DataAdded    int64  `json:"data_added"`
			FilesNew     int    `json:"files_new"`
			FilesChanged int    `json:"files_changed"`
		}
		if err := json.Unmarshal(line, &msg); err != nil {
			continue
		}
		if msg.MessageType == "summary" {
			result = &BackupResult{
				SnapshotID:   msg.SnapshotID,
				BytesAdded:   msg.DataAdded,
				FilesNew:     msg.FilesNew,
				FilesChanged: msg.FilesChanged,
			}
		}
	}

	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("restic backup: %w", err)
	}
	if result == nil {
		return nil, fmt.Errorf("restic backup: no summary in output")
	}
	return result, nil
}

// cmd builds an exec.Cmd for the given restic subcommand with repo and password set.
func (r *Runner) cmd(ctx context.Context, opts BackupOptions, args ...string) *exec.Cmd {
	all := append([]string{"-r", opts.Repo}, args...)
	cmd := exec.CommandContext(ctx, r.bin, all...)
	cmd.Env = append(cmd.Environ(), "RESTIC_PASSWORD="+opts.Password)
	return cmd
}
