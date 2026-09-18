//go:build linux || darwin

package scheduler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteUnits(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "units")
	paths, err := writeUnits(dir, Options{Binary: "/opt/botcheck/botcheck", At: "03:17"})
	if err != nil {
		t.Fatalf("writeUnits: %v", err)
	}
	if len(paths) == 0 {
		t.Fatal("writeUnits wrote no files")
	}
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Errorf("read %s: %v", p, err)
			continue
		}
		if !strings.Contains(string(b), "/opt/botcheck/botcheck") {
			t.Errorf("%s: missing binary path", p)
		}
	}
}
