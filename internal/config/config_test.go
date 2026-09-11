package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	content := []byte(`
cache_dir: "/tmp/cache"
sources:
  - name: googlebot
    category: search
    type: http
    url: https://developers.google.com/static/search/apis/ipranges/googlebot.json
    enabled: true
`)
	f, err := os.CreateTemp("", "botcheck-test-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	if _, err := f.Write(content); err != nil {
		t.Fatal(err)
	}
	f.Close()

	cfg, err := Load(f.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.CacheDir != "/tmp/cache" {
		t.Errorf("expected /tmp/cache, got %s", cfg.CacheDir)
	}

	if len(cfg.Sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(cfg.Sources))
	}

	if cfg.Sources[0].Name != "googlebot" {
		t.Errorf("expected googlebot, got %s", cfg.Sources[0].Name)
	}
}
