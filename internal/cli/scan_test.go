package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/saurabhsharma2u/iambot/internal/matcher"
	"github.com/saurabhsharma2u/iambot/internal/registry"
)

func TestScanCmd(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", tmp)

	reg := registry.NewDiskRegistry("")
	entries := []registry.Entry{
		{Prefix: "1.1.1.1/32", Meta: matcher.Meta{Source: "test", Category: "test", Name: "test"}},
	}
	stats := registry.Stats{
		TotalPrefixes: 1,
		Sources:       map[string]int{"test": 1},
		Categories:    map[string]int{"test": 1},
	}
	if err := reg.SaveRaw(context.Background(), entries, stats); err != nil {
		t.Fatal(err)
	}

	longUA := strings.Repeat("A", 100*1024)
	logContent := "1.1.1.1 - -\n2.2.2.2 - - [11/Sep/2026:00:00:00 +0000] \"GET / HTTP/1.1\" 200 10 \"-\" \"" + longUA + "\"\n"
	logFile := filepath.Join(tmp, "test.log")
	err := os.WriteFile(logFile, []byte(logContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	oldStdout := os.Stdout
	rOut, wOut, _ := os.Pipe()
	os.Stdout = wOut
	defer func() {
		os.Stdout = oldStdout
	}()

	rootCmd.SetArgs([]string{"scan", logFile, "--quiet", "--format", "auto", "--output", "text"})
	defer rootCmd.SetArgs(nil)

	err = rootCmd.ExecuteContext(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_ = wOut.Close()
	var bufOut bytes.Buffer
	_, _ = io.Copy(&bufOut, rOut)

	out := bufOut.String()
	if !strings.Contains(out, "1.1.1.1") {
		t.Errorf("expected output to contain 1.1.1.1, got %q", out)
	}
}

func TestScanCmdMissingFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", tmp)

	reg := registry.NewDiskRegistry("")
	entries := []registry.Entry{
		{Prefix: "1.1.1.1/32", Meta: matcher.Meta{Source: "test"}},
	}
	stats := registry.Stats{TotalPrefixes: 1, Sources: map[string]int{"test": 1}}
	if err := reg.SaveRaw(context.Background(), entries, stats); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"scan", filepath.Join(tmp, "does-not-exist.log"), "--quiet"})
	defer rootCmd.SetArgs(nil)

	if err := rootCmd.ExecuteContext(context.Background()); err == nil {
		t.Error("expected error for missing log file, got nil")
	}
}
