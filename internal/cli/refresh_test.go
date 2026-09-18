package cli

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/saurabhsharma2u/botcheck/internal/registry"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func mustParseAddr(t *testing.T, s string) netip.Addr {
	t.Helper()
	ip, err := netip.ParseAddr(s)
	if err != nil {
		t.Fatalf("parse addr %q: %v", s, err)
	}
	return ip
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

func TestRunRefreshLockContention(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache")

	release, err := acquireCacheLock(cacheDir)
	if err != nil {
		t.Fatalf("acquire lock: %v", err)
	}
	defer release()

	writeFile(t, filepath.Join(dir, "local.txt"), "9.9.9.9\n")
	cfgPath := filepath.Join(dir, "botcheck.yaml")
	writeFile(t, cfgPath, fmt.Sprintf("cache_dir: %q\nregistry_url: \"off\"\nsources:\n  - name: extra\n    category: test\n    type: file\n    path: %s\n    enabled: true\n",
		cacheDir, filepath.Join(dir, "local.txt")))

	if err := runRefresh(context.Background(), cfgPath, io.Discard, io.Discard); err == nil {
		t.Fatal("expected lock contention error, got nil")
	}
}

func TestRunRefreshStampsVerifySuffixes(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache")
	writeFile(t, filepath.Join(dir, "local.txt"), "9.9.9.0/24\n")
	cfgPath := filepath.Join(dir, "botcheck.yaml")
	writeFile(t, cfgPath, fmt.Sprintf("cache_dir: %q\nregistry_url: \"off\"\nsources:\n  - name: extra\n    category: test\n    type: file\n    path: %s\n    enabled: true\n    verify_suffixes: [\"example.com\", \"example.net\"]\n",
		cacheDir, filepath.Join(dir, "local.txt")))

	if err := runRefresh(context.Background(), cfgPath, io.Discard, io.Discard); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	reg := registry.NewDiskRegistry(cacheDir)
	if err := reg.Load(context.Background()); err != nil {
		t.Fatalf("load registry: %v", err)
	}
	hit, ok := reg.Contains(mustParseAddr(t, "9.9.9.9"))
	if !ok {
		t.Fatal("expected 9.9.9.9 to match")
	}
	if got := hit.Meta.Extra["verify_suffixes"]; got != "example.com,example.net" {
		t.Errorf("expected stamped suffixes, got %q", got)
	}

	bySource, err := registry.LoadVerifyMap(cacheDir)
	if err != nil {
		t.Fatalf("load verify map: %v", err)
	}
	if len(bySource["extra"]) != 2 || bySource["extra"][0] != "example.com" {
		t.Errorf("unexpected verify map: %v", bySource)
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

func TestRunRefreshDNSOnlySource(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache")
	writeFile(t, filepath.Join(dir, "local.txt"), "9.9.9.0/24\n")
	cfgPath := filepath.Join(dir, "botcheck.yaml")
	writeFile(t, cfgPath, fmt.Sprintf("cache_dir: %q\nregistry_url: \"off\"\nsources:\n"+
		"  - name: extra\n    category: test\n    type: file\n    path: %s\n    enabled: true\n"+
		"  - name: sogou\n    category: search\n    type: dns\n    enabled: true\n    verify_suffixes: [\"sogou.com\"]\n",
		cacheDir, filepath.Join(dir, "local.txt")))

	if err := runRefresh(context.Background(), cfgPath, io.Discard, io.Discard); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	bySource, err := registry.LoadVerifyMap(cacheDir)
	if err != nil {
		t.Fatalf("load verify map: %v", err)
	}
	if len(bySource["sogou"]) != 1 || bySource["sogou"][0] != "sogou.com" {
		t.Errorf("dns-only source missing from verify map: %v", bySource)
	}
	// Prefix data must be unaffected by the dns-only entry.
	if stats := loadStats(t, cacheDir); stats.TotalPrefixes != 1 {
		t.Errorf("expected 1 prefix, got %d", stats.TotalPrefixes)
	}
}

func TestRunRefreshDNSOnlyWritesVerifyMap(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache")
	cfgPath := filepath.Join(dir, "botcheck.yaml")
	writeFile(t, cfgPath, fmt.Sprintf("cache_dir: %q\nregistry_url: \"off\"\nsources:\n"+
		"  - name: sogou\n    category: search\n    type: dns\n    enabled: true\n    verify_suffixes: [\"sogou.com\"]\n",
		cacheDir))

	if err := runRefresh(context.Background(), cfgPath, io.Discard, io.Discard); err != nil {
		t.Fatalf("dns-only refresh must succeed, got: %v", err)
	}
	bySource, err := registry.LoadVerifyMap(cacheDir)
	if err != nil {
		t.Fatalf("load verify map: %v", err)
	}
	if len(bySource["sogou"]) != 1 {
		t.Errorf("unexpected verify map: %v", bySource)
	}
}
