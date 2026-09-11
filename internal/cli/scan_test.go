package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScanCmd(t *testing.T) {
	tmp := t.TempDir()

	logFile := filepath.Join(tmp, "test.log")
	err := os.WriteFile(logFile, []byte("1.1.1.1 - -\n2.2.2.2 - -\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	oldStdout := os.Stdout
	rOut, wOut, _ := os.Pipe()
	os.Stdout = wOut
	defer func() {
		os.Stdout = oldStdout
	}()

	rootCmd.SetArgs([]string{"scan", logFile, "--quiet"})

	err = rootCmd.ExecuteContext(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wOut.Close()
	var bufOut bytes.Buffer
	_, _ = io.Copy(&bufOut, rOut)

	out := bufOut.String()
	if !strings.Contains(out, "1.1.1.1") {
		t.Errorf("expected output to contain 1.1.1.1, got %q", out)
	}
}
