package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SaveToken writes the token to path with owner-only permissions (0600).
// The parent directory is created if it does not exist.
func SaveToken(path, token string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create token dir: %w", err)
	}
	if err := os.WriteFile(path, []byte(token), 0600); err != nil {
		return fmt.Errorf("write token file: %w", err)
	}
	return nil
}

// LoadToken reads the token from path and trims whitespace.
// Returns an empty string and no error if the file does not exist.
func LoadToken(path string) (string, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read token file: %w", err)
	}
	return strings.TrimSpace(string(b)), nil
}
