package cli

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/saurabhsharma2u/iambot/internal/registry"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func feedServer(prefixes ...string) *httptest.Server {
	var b strings.Builder
	b.WriteString(`{"prefixes": [`)
	for i, p := range prefixes {
		if i > 0 {
			b.WriteString(",")
		}
		fmt.Fprintf(&b, `{"ipv4Prefix": %q}`, p)
	}
	b.WriteString(`]}`)
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(b.String()))
	}))
}

func loadStats(t *testing.T, cacheDir string) registry.Stats {
	t.Helper()
	reg := registry.NewDiskRegistry(cacheDir)
	if err := reg.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	return reg.Stats()
}

func TestRunRefreshRegistryMerge(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache")

	good := feedServer("192.0.2.0/24", "198.51.100.0/24")
	defer good.Close()
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	badURL := bad.URL
	bad.Close()

	manifest := fmt.Sprintf("version: \"test-9\"\nsources:\n"+
		"  - name: good\n    category: search\n    type: http\n    url: %s\n    enabled: true\n"+
		"  - name: bad\n    category: search\n    type: http\n    url: %s\n    enabled: true\n",
		good.URL, badURL)
	manServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(manifest))
	}))
	defer manServer.Close()

	writeFile(t, filepath.Join(dir, "local.txt"), "9.9.9.9\n")
	writeFile(t, filepath.Join(dir, "override.txt"), "8.8.8.8\n")
	cfg := fmt.Sprintf("cache_dir: %q\nregistry_url: %q\nsources:\n"+
		"  - name: good\n    category: local\n    type: file\n    path: %s\n    enabled: true\n"+
		"  - name: extra\n    category: test\n    type: file\n    path: %s\n    enabled: true\n",
		cacheDir, manServer.URL, filepath.Join(dir, "override.txt"), filepath.Join(dir, "local.txt"))
	cfgPath := filepath.Join(dir, "botcheck.yaml")
	writeFile(t, cfgPath, cfg)

	if err := runRefresh(context.Background(), cfgPath, io.Discard, io.Discard); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stats := loadStats(t, cacheDir)
	if stats.TotalPrefixes != 2 {
		t.Errorf("expected 2 prefixes (local override + extra), got %d", stats.TotalPrefixes)
	}
	if stats.DataVersion != "test-9" {
		t.Errorf("expected data version test-9, got %q", stats.DataVersion)
	}
	if stats.Sources["good"] != 1 || stats.Sources["extra"] != 1 {
		t.Errorf("unexpected per-source stats: %v", stats.Sources)
	}
	if _, ok := stats.Sources["bad"]; ok {
		t.Errorf("failing source must not appear in stats: %v", stats.Sources)
	}
	if stats.Categories["local"] != 1 {
		t.Errorf("expected local override to win (category local), got %v", stats.Categories)
	}
}

func TestRunRefreshOfflineKeepsLastGood(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache")

	good := feedServer("192.0.2.0/24")
	defer good.Close()
	manifest := fmt.Sprintf("version: \"v1\"\nsources:\n  - name: good\n    category: search\n    type: http\n    url: %s\n    enabled: true\n", good.URL)
	manServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(manifest))
	}))
	cfgPath := filepath.Join(dir, "botcheck.yaml")
	writeFile(t, cfgPath, fmt.Sprintf("cache_dir: %q\nregistry_url: %q\n", cacheDir, manServer.URL))

	if err := runRefresh(context.Background(), cfgPath, io.Discard, io.Discard); err != nil {
		t.Fatalf("seed update failed: %v", err)
	}
	before := loadStats(t, cacheDir)
	if before.TotalPrefixes != 1 {
		t.Fatalf("expected 1 seeded prefix, got %d", before.TotalPrefixes)
	}

	manServer.Close()
	if err := runRefresh(context.Background(), cfgPath, io.Discard, io.Discard); err != nil {
		t.Fatalf("expected nil error keeping last-good, got %v", err)
	}
	after := loadStats(t, cacheDir)
	if after.TotalPrefixes != before.TotalPrefixes || after.DataVersion != before.DataVersion {
		t.Errorf("cache changed: before %+v, after %+v", before, after)
	}
}

func TestRunRefreshOfflineEmptyCacheErrors(t *testing.T) {
	dir := t.TempDir()
	dead := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	deadURL := dead.URL
	dead.Close()

	cfgPath := filepath.Join(dir, "botcheck.yaml")
	writeFile(t, cfgPath, fmt.Sprintf("cache_dir: %q\nregistry_url: %q\n", filepath.Join(dir, "cache"), deadURL))

	err := runRefresh(context.Background(), cfgPath, io.Discard, io.Discard)
	if err == nil {
		t.Fatal("expected error with unreachable registry and empty cache, got nil")
	}
}

func TestRunRefreshRegistryOff(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache")
	writeFile(t, filepath.Join(dir, "local.txt"), "9.9.9.9\n")
	cfgPath := filepath.Join(dir, "botcheck.yaml")
	writeFile(t, cfgPath, fmt.Sprintf("cache_dir: %q\nregistry_url: \"off\"\nsources:\n  - name: extra\n    category: test\n    type: file\n    path: %s\n    enabled: true\n",
		cacheDir, filepath.Join(dir, "local.txt")))

	if err := runRefresh(context.Background(), cfgPath, io.Discard, io.Discard); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	stats := loadStats(t, cacheDir)
	if stats.TotalPrefixes != 1 || stats.DataVersion != "" {
		t.Errorf("unexpected stats: %+v", stats)
	}
}
