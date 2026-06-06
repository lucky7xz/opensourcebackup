package restic

import (
	"path/filepath"
	"testing"
)

func TestResolveResticBin_AbsolutePassesThrough(t *testing.T) {
	abs, err := filepath.Abs(filepath.Join("opt", "restic"))
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	if got := resolveResticBin(abs); got != abs {
		t.Errorf("absolute path must pass through unchanged: got %q want %q", got, abs)
	}
}

// The core guarantee: a bare or empty reference must never come back as a path
// that os/exec would refuse as relative-to-cwd (exec.ErrDot). It is acceptable
// to return either an absolute path or the unchanged bare name (PATH fallback),
// but never a relative path like ".\\restic.exe".
func TestResolveResticBin_NeverRelativePath(t *testing.T) {
	for _, in := range []string{"", "restic"} {
		got := resolveResticBin(in)
		if got == "" {
			t.Fatalf("resolveResticBin(%q) returned empty string", in)
		}
		if got != "restic" && !filepath.IsAbs(got) {
			t.Errorf("resolveResticBin(%q) = %q: want absolute path or bare fallback, got relative", in, got)
		}
	}
}
