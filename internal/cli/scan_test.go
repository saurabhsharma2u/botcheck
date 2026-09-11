package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/saurabhsharma2u/botcheck/internal/matcher"
	"github.com/saurabhsharma2u/botcheck/internal/registry"
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

	rootCmd.SetArgs([]string{"scan", logFile, "--quiet", "--input", "auto", "--output", "text"})
	defer rootCmd.SetArgs(nil)

	var bufOut bytes.Buffer
	drained := make(chan struct{})
	go func() {
		_, _ = io.Copy(&bufOut, rOut)
		close(drained)
	}()

	err = rootCmd.ExecuteContext(context.Background())
	_ = wOut.Close()
	<-drained
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

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

func TestScanCmdStdin(t *testing.T) {
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

	inFile := filepath.Join(tmp, "in.log")
	if err := os.WriteFile(inFile, []byte("1.1.1.1 - -\n"), 0644); err != nil {
		t.Fatal(err)
	}
	in, err := os.Open(inFile)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = in.Close() }()

	oldStdin, oldStdout := os.Stdin, os.Stdout
	rOut, wOut, _ := os.Pipe()
	os.Stdin, os.Stdout = in, wOut
	defer func() {
		os.Stdin, os.Stdout = oldStdin, oldStdout
	}()

	rootCmd.SetArgs([]string{"scan", "-", "--quiet"})
	defer rootCmd.SetArgs(nil)

	var bufOut bytes.Buffer
	drained := make(chan struct{})
	go func() {
		_, _ = io.Copy(&bufOut, rOut)
		close(drained)
	}()

	if err := rootCmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = wOut.Close()
	<-drained

	if out := bufOut.String(); !strings.Contains(out, "1.1.1.1") {
		t.Errorf("expected output to contain 1.1.1.1, got %q", out)
	}
}
