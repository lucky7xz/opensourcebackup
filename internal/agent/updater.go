package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

// UpdateInfo is returned by the control plane heartbeat.
type UpdateInfo struct {
	UpdateAvailable    bool   `json:"update_available"`
	RecommendedVersion string `json:"recommended_version"`
	DownloadURL        string `json:"update_download_url"`
	ExpectedChecksum   string `json:"update_checksum_sha256"` // hex SHA-256
}

// Updater handles safe agent binary self-updates.
//
// Safety guarantees:
//  1. Checksum is verified before replacing the binary.
//  2. The new binary is written to a temp file first.
//  3. The old binary is kept as a backup (.bak) for manual rollback.
//  4. Auto-update is disabled if ExpectedChecksum is empty.
type Updater struct {
	currentVersion string
	binaryPath     string
	log            *slog.Logger
	client         *http.Client
}

// NewUpdater creates an Updater for the given binary path.
func NewUpdater(currentVersion, binaryPath string, log *slog.Logger) *Updater {
	return &Updater{
		currentVersion: currentVersion,
		binaryPath:     binaryPath,
		log:            log,
		client:         &http.Client{},
	}
}

// ShouldUpdate returns true if an update should be applied.
// Requires both UpdateAvailable=true AND a non-empty checksum (safety gate).
func (u *Updater) ShouldUpdate(info UpdateInfo) bool {
	return info.UpdateAvailable &&
		info.RecommendedVersion != u.currentVersion &&
		info.DownloadURL != "" &&
		info.ExpectedChecksum != ""
}

// Apply downloads, verifies, and installs the update.
// Never called automatically — must be explicitly invoked.
func (u *Updater) Apply(ctx context.Context, info UpdateInfo) error {
	if info.ExpectedChecksum == "" {
		return fmt.Errorf("updater: refusing update without checksum — unsafe")
	}

	u.log.Info("agent update: downloading", "version", info.RecommendedVersion, "url", info.DownloadURL)

	// Download to temp file
	tmpPath := u.binaryPath + ".new"
	if err := u.download(ctx, info.DownloadURL, tmpPath); err != nil {
		os.Remove(tmpPath) //nolint:errcheck
		return fmt.Errorf("download: %w", err)
	}

	// Verify checksum
	if err := u.verifyChecksum(tmpPath, info.ExpectedChecksum); err != nil {
		os.Remove(tmpPath) //nolint:errcheck
		return fmt.Errorf("checksum mismatch: %w", err)
	}
	u.log.Info("agent update: checksum verified")

	// Make executable
	if err := os.Chmod(tmpPath, 0755); err != nil {
		os.Remove(tmpPath) //nolint:errcheck
		return fmt.Errorf("chmod: %w", err)
	}

	// Backup old binary
	bakPath := u.binaryPath + ".bak"
	if err := os.Rename(u.binaryPath, bakPath); err != nil {
		u.log.Warn("agent update: could not backup old binary", "error", err)
	}

	// Replace binary
	if err := os.Rename(tmpPath, u.binaryPath); err != nil {
		// Try to restore backup
		os.Rename(bakPath, u.binaryPath) //nolint:errcheck
		return fmt.Errorf("replace binary: %w", err)
	}

	u.log.Info("agent update: installed successfully", "version", info.RecommendedVersion)
	u.log.Info("agent update: restart required — service manager will handle this")
	return nil
}

// Platform returns the current OS/arch platform string for download URL construction.
func Platform() string {
	return runtime.GOOS + "-" + runtime.GOARCH
}

func (u *Updater) download(ctx context.Context, url, dest string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := u.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned %d", resp.StatusCode)
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0700); err != nil {
		return err
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

func (u *Updater) verifyChecksum(path, expected string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	got := hex.EncodeToString(h.Sum(nil))
	if got != expected {
		return fmt.Errorf("got %s, expected %s", got, expected)
	}
	return nil
}
