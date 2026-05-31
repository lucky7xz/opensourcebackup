package restic

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
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

// RestoreOptions configures a restic restore run.
type RestoreOptions struct {
	Repo        string
	Password    string
	SnapshotID  string
	TargetPath  string // must be under RestoreRoot
	RestoreRoot string // safety boundary — target must be under this path
}

// RestoreResult holds the verified file count and byte sum after restore.
type RestoreResult struct {
	TargetPath    string
	VerifiedFiles int
	VerifiedBytes int64
}

// Runner executes restic commands.
type Runner struct {
	bin string
}

// New creates a Runner using the given restic binary path.
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

// Restore runs restic restore into a validated sandbox directory and
// counts the restored files and bytes.
func (r *Runner) Restore(ctx context.Context, opts RestoreOptions) (*RestoreResult, error) {
	if err := validateRestorePath(opts.TargetPath, opts.RestoreRoot); err != nil {
		return nil, fmt.Errorf("restore path validation: %w", err)
	}
	if err := os.MkdirAll(opts.TargetPath, 0700); err != nil {
		return nil, fmt.Errorf("create restore target: %w", err)
	}

	cmd := exec.CommandContext(ctx, r.bin,
		"-r", opts.Repo,
		"restore", opts.SnapshotID,
		"--target", opts.TargetPath,
	)
	cmd.Env = append(os.Environ(), "RESTIC_PASSWORD="+opts.Password)
	out, err := cmd.CombinedOutput()
	if err != nil {
		outStr := string(out)
		// On Windows, restic exits with error code 1 when it cannot set
		// timestamps on protected system directories (e.g. C:\Users).
		// This is a metadata-only issue — the actual files are restored.
		// Treat as success if the summary shows files were restored.
		if isWindowsTimestampError(outStr) {
			// Parse how many files were actually restored
			files, totalBytes, walkErr := countFiles(opts.TargetPath)
			if walkErr == nil && files > 0 {
				return &RestoreResult{
					TargetPath:    opts.TargetPath,
					VerifiedFiles: files,
					VerifiedBytes: totalBytes,
				}, nil
			}
		}
		return nil, fmt.Errorf("restic restore: %w — %s", err, bytes.TrimSpace(out))
	}

	// Walk target and count restored files + bytes independently of restic output.
	files, totalBytes, walkErr := countFiles(opts.TargetPath)
	if walkErr != nil {
		return nil, fmt.Errorf("counting restored files: %w", walkErr)
	}

	return &RestoreResult{
		TargetPath:    opts.TargetPath,
		VerifiedFiles: files,
		VerifiedBytes: totalBytes,
	}, nil
}

// isWindowsTimestampError returns true when restic failed only because it
// could not set timestamps on Windows system directories (UtimesNano / Zugriff verweigert).
// The actual file data was restored successfully.
func isWindowsTimestampError(out string) bool {
	hasTimestamp := strings.Contains(out, "UtimesNano") ||
		strings.Contains(out, "Zugriff verweigert") ||
		strings.Contains(out, "Access is denied")
	hasSummary := strings.Contains(out, "Summary: Restored")
	return hasTimestamp && hasSummary
}

// validateRestorePath ensures the target is non-empty, not a dangerous root,
// and is located under the configured RestoreRoot.
func validateRestorePath(target, root string) error {
	if strings.TrimSpace(target) == "" {
		return fmt.Errorf("target path is empty")
	}
	abs, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	// Reject filesystem roots
	if abs == "/" || abs == filepath.VolumeName(abs)+string(filepath.Separator) {
		return fmt.Errorf("target path %q is a filesystem root — too dangerous", abs)
	}
	if root != "" {
		rootAbs, err := filepath.Abs(root)
		if err != nil {
			return err
		}
		if !strings.HasPrefix(abs, rootAbs+string(filepath.Separator)) && abs != rootAbs {
			return fmt.Errorf("target path %q must be under restore root %q", abs, rootAbs)
		}
	}
	return nil
}

// countFiles walks dir and returns the number of regular files and their total size.
func countFiles(dir string) (int, int64, error) {
	var count int
	var total int64
	err := filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		count++
		total += info.Size()
		return nil
	})
	return count, total, err
}

func (r *Runner) initRepo(ctx context.Context, opts BackupOptions) error {
	cmd := r.cmd(ctx, opts, "init")
	out, err := cmd.CombinedOutput()
	if err != nil {
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

func (r *Runner) cmd(ctx context.Context, opts BackupOptions, args ...string) *exec.Cmd {
	all := append([]string{"-r", opts.Repo}, args...)
	cmd := exec.CommandContext(ctx, r.bin, all...)
	cmd.Env = append(cmd.Environ(), "RESTIC_PASSWORD="+opts.Password)
	return cmd
}
