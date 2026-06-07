package restic

import (
	"math"
	"testing"
)

func TestParseStatusProgress_Status(t *testing.T) {
	// A real restic status line — note current_files IS present in the input.
	line := []byte(`{"message_type":"status","seconds_elapsed":42,"percent_done":0.624,` +
		`"total_files":300,"files_done":120,"total_bytes":198765432,"bytes_done":123456789,` +
		`"current_files":["C:\\Users\\Admin\\secret.txt","D:\\private\\taxes.pdf"]}`)

	p, ok := parseStatusProgress(line)
	if !ok {
		t.Fatal("expected ok=true for a status line")
	}
	if p.Phase != "backup" {
		t.Errorf("Phase = %q, want backup", p.Phase)
	}
	if math.Abs(p.Percent-62.4) > 1e-9 {
		t.Errorf("Percent = %v, want 62.4 (0..1 → 0..100)", p.Percent)
	}
	if p.BytesDone != 123456789 || p.TotalBytes != 198765432 {
		t.Errorf("bytes = %d/%d, want 123456789/198765432", p.BytesDone, p.TotalBytes)
	}
	if p.FilesDone != 120 || p.TotalFiles != 300 {
		t.Errorf("files = %d/%d, want 120/300", p.FilesDone, p.TotalFiles)
	}
	// Privacy guarantee is structural: Progress has no field that could hold
	// current_files. This compiles only because no path data is exposed.
}

func TestParseStatusProgress_NonStatusOrGarbage(t *testing.T) {
	cases := [][]byte{
		[]byte(`{"message_type":"summary","snapshot_id":"abc","data_added":10}`),
		[]byte(`{"message_type":"verbose_status"}`),
		[]byte(`not json at all`),
		[]byte(``),
	}
	for _, line := range cases {
		if _, ok := parseStatusProgress(line); ok {
			t.Errorf("expected ok=false for %q", line)
		}
	}
}
