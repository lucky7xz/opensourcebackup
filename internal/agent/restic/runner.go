package restic

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// BackupOptions configures a restic backup run.
type BackupOptions struct {
	Repo               string
	Password           string
	Includes           []string
	Excludes           []string
	Tags               []string
	BandwidthLimitKbps int // 0 = unlimited; passed as --limit-upload N
	// OnProgress, if set, is called for each restic progress update during the
	// backup. Optional (nil = no progress reporting). It carries aggregate
	// counters only — never file paths.
	OnProgress func(Progress)
}

// Progress is a single live progress update parsed from restic's --json output.
// Deliberately no file paths (restic's current_files is dropped) — privacy.
type Progress struct {
	Phase      string  // "backup"
	Percent    float64 // 0..100
	BytesDone  int64
	TotalBytes int64
	FilesDone  int
	TotalFiles int
}

// VerifyOptions configures a restic check run.
type VerifyOptions struct {
	Repo     string
	Password string
	ReadData bool // --read-data: verify actual file contents (slow but thorough)
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
	return &Runner{bin: resolveResticBin(bin)}
}

// resolveResticBin turns a possibly empty or relative restic reference into an
// absolute path. Go's os/exec refuses to run a binary resolved relative to the
// current directory (exec.ErrDot), which breaks the common product layout where
// restic.exe sits next to the agent and the working directory is the install
// dir. Resolving to an absolute path up front avoids that refusal.
func resolveResticBin(bin string) string {
	if bin == "" {
		bin = "restic"
	}
	if filepath.IsAbs(bin) {
		return bin
	}
	// Prefer a restic binary bundled next to the agent executable.
	if exe, err := os.Executable(); err == nil {
		cand := filepath.Join(filepath.Dir(exe), bin)
		if runtime.GOOS == "windows" && filepath.Ext(cand) == "" {
			cand += ".exe"
		}
		if _, statErr := os.Stat(cand); statErr == nil {
			return cand
		}
	}
	// Fall back to PATH; resolve to absolute so a match in the working
	// directory (exec.ErrDot) is still usable instead of being refused.
	if p, err := exec.LookPath(bin); (err == nil || errors.Is(err, exec.ErrDot)) && p != "" {
		if abs, absErr := filepath.Abs(p); absErr == nil {
			return abs
		}
		return p
	}
	return bin
}

// Backup initializes the repository if needed, then runs a backup.
//
// Before backing up it removes stale locks. A crash or hard power-off leaves the
// previous run's lock behind, which otherwise blocks EVERY future backup with
// "repository is already locked" — the agent could never recover unattended.
// Unlock only removes dead/old locks (see Unlock), so a live lock held by
// another host backing up the same repo is preserved.
func (r *Runner) Backup(ctx context.Context, opts BackupOptions) (*BackupResult, error) {
	// Best-effort: if this fails the backup below fails loudly and is logged.
	_ = r.Unlock(ctx, opts.Repo, opts.Password)
	if err := r.initRepo(ctx, opts); err != nil {
		return nil, err
	}
	return r.runBackup(ctx, opts)
}

// Unlock removes stale locks from the repository — locks whose creating process
// has exited (same host, dead PID) or that are older than restic's stale
// timeout. It deliberately does NOT use --remove-all: a fresh lock from another
// host currently backing up the same repository (restic refreshes live locks
// every few minutes) is therefore left intact.
func (r *Runner) Unlock(ctx context.Context, repo, password string) error {
	args := []string{"-r", repo, "unlock"}
	if password == "" {
		args = append(args, "--insecure-no-password")
	}
	cmd := exec.CommandContext(ctx, r.bin, args...)
	cmd.Env = append(os.Environ(), "RESTIC_PASSWORD="+password)
	tuneResticProcess(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("restic unlock: %w — %s", err, bytes.TrimSpace(out))
	}
	return nil
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
	tuneResticProcess(cmd)
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
	cmd := r.cmd(ctx, opts, "init", "--quiet")
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

// Verify runs restic check to verify repository integrity without a full restore.
func (r *Runner) Verify(ctx context.Context, opts VerifyOptions) error {
	args := []string{"check"}
	if opts.ReadData {
		args = append(args, "--read-data")
	}
	if opts.Password == "" {
		args = append(args, "--insecure-no-password")
	}
	cmd := exec.CommandContext(ctx, r.bin, append([]string{"-r", opts.Repo}, args...)...)
	cmd.Env = append(os.Environ(), "RESTIC_PASSWORD="+opts.Password)
	tuneResticProcess(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("restic check: %w — %s", err, bytes.TrimSpace(out))
	}
	return nil
}

// parseStatusProgress parses a restic --json "status" line into a Progress.
// ok is false for any non-status or malformed line. It decodes ONLY aggregate
// counters; restic's current_files (file paths) is never read or returned —
// privacy / data minimisation (DSGVO).
func parseStatusProgress(line []byte) (Progress, bool) {
	var msg struct {
		MessageType string  `json:"message_type"`
		PercentDone float64 `json:"percent_done"` // 0..1
		TotalFiles  int     `json:"total_files"`
		FilesDone   int     `json:"files_done"`
		TotalBytes  int64   `json:"total_bytes"`
		BytesDone   int64   `json:"bytes_done"`
	}
	if err := json.Unmarshal(line, &msg); err != nil || msg.MessageType != "status" {
		return Progress{}, false
	}
	return Progress{
		Phase:      "backup",
		Percent:    msg.PercentDone * 100, // restic 0..1 → 0..100
		BytesDone:  msg.BytesDone,
		TotalBytes: msg.TotalBytes,
		FilesDone:  msg.FilesDone,
		TotalFiles: msg.TotalFiles,
	}, true
}

func (r *Runner) runBackup(ctx context.Context, opts BackupOptions) (*BackupResult, error) {
	args := append([]string{"backup", "--json"}, opts.Includes...)
	for _, ex := range opts.Excludes {
		args = append(args, "--exclude", ex)
	}
	for _, tag := range opts.Tags {
		args = append(args, "--tag", tag)
	}
	if opts.BandwidthLimitKbps > 0 {
		args = append(args, "--limit-upload", fmt.Sprintf("%d", opts.BandwidthLimitKbps))
	}

	cmd := r.cmd(ctx, opts, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("restic backup pipe: %w", err)
	}
	// Capture stderr so a failing backup reports restic's actual message
	// (repository unreachable, password, lock, …) instead of a bare exit code.
	// Bounded to avoid unbounded memory if restic floods stderr.
	var stderr limitedBuffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("restic backup start: %w", err)
	}

	var result *BackupResult
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Bytes()
		if p, ok := parseStatusProgress(line); ok {
			if opts.OnProgress != nil {
				opts.OnProgress(p)
			}
			continue
		}
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
		// Exit status 3: restic completed but some files could not be read
		// (e.g. locked files on Windows, permission denied on system dirs).
		// A snapshot WAS created — treat as partial success if we got a summary.
		if isExitStatus(err, 3) && result != nil {
			return result, nil
		}
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return nil, fmt.Errorf("restic backup: %w — %s", err, msg)
		}
		return nil, fmt.Errorf("restic backup: %w", err)
	}
	if result == nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return nil, fmt.Errorf("restic backup: no summary in output — %s", msg)
		}
		return nil, fmt.Errorf("restic backup: no summary in output")
	}
	return result, nil
}

// limitedBuffer is an io.Writer that retains at most maxStderrBytes, so a
// misbehaving restic flooding stderr can't exhaust agent memory. Excess is
// silently dropped — we only need the message for diagnostics.
type limitedBuffer struct {
	buf bytes.Buffer
}

const maxStderrBytes = 8 << 10 // 8 KiB is plenty for an error message

func (l *limitedBuffer) Write(p []byte) (int, error) {
	if remaining := maxStderrBytes - l.buf.Len(); remaining > 0 {
		if len(p) > remaining {
			l.buf.Write(p[:remaining])
		} else {
			l.buf.Write(p)
		}
	}
	return len(p), nil // always report full write so restic isn't blocked
}

func (l *limitedBuffer) String() string { return l.buf.String() }

// isExitStatus returns true if err is an *exec.ExitError with the given exit code.
func isExitStatus(err error, code int) bool {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode() == code
	}
	return false
}

func (r *Runner) cmd(ctx context.Context, opts BackupOptions, args ...string) *exec.Cmd {
	all := append([]string{"-r", opts.Repo}, args...)
	if opts.Password == "" {
		all = append(all, "--insecure-no-password")
	}
	cmd := exec.CommandContext(ctx, r.bin, all...)
	cmd.Env = append(cmd.Environ(), "RESTIC_PASSWORD="+opts.Password)
	tuneResticProcess(cmd)
	return cmd
}
