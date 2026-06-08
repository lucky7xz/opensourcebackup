package restic

import (
	"strings"
	"testing"
)

func TestLimitedBuffer_RetainsUpToCap(t *testing.T) {
	var b limitedBuffer
	in := "repository unreachable: connection refused"
	n, err := b.Write([]byte(in))
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if n != len(in) {
		t.Errorf("Write must report full length so restic isn't blocked: want %d, got %d", len(in), n)
	}
	if b.String() != in {
		t.Errorf("want %q, got %q", in, b.String())
	}
}

func TestLimitedBuffer_CapsAtMax(t *testing.T) {
	var b limitedBuffer
	// Write well beyond the cap in several chunks.
	flood := strings.Repeat("x", maxStderrBytes*3)
	for i := 0; i < len(flood); i += 4096 {
		end := i + 4096
		if end > len(flood) {
			end = len(flood)
		}
		n, err := b.Write([]byte(flood[i:end]))
		if err != nil {
			t.Fatalf("Write: %v", err)
		}
		if n != end-i {
			t.Errorf("Write must always report full chunk length: want %d, got %d", end-i, n)
		}
	}
	if got := len(b.String()); got != maxStderrBytes {
		t.Errorf("buffer must be capped at %d bytes, got %d", maxStderrBytes, got)
	}
}

func TestLimitedBuffer_EmptyByDefault(t *testing.T) {
	var b limitedBuffer
	if b.String() != "" {
		t.Errorf("zero-value buffer must be empty, got %q", b.String())
	}
}
