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
	defer func() { _ = os.Remove(f.Name()) }()

	if _, err := f.Write(content); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

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

func TestLoadImports(t *testing.T) {
	dir := t.TempDir()

	extra := `
sources:
  - name: imported
    category: search
    type: http
    url: https://example.com/imported.json
    enabled: true
  - name: overridden
    category: search
    type: http
    url: https://example.com/old.json
    enabled: false
`
	if err := os.WriteFile(dir+"/extra.yaml", []byte(extra), 0644); err != nil {
		t.Fatal(err)
	}

	main := `
cache_dir: "/tmp/cache"
imports:
  - extra.yaml
sources:
  - name: overridden
    category: ai
    type: http
    url: https://example.com/new.json
    enabled: true
`
	mainPath := dir + "/botcheck.yaml"
	if err := os.WriteFile(mainPath, []byte(main), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(mainPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Sources) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(cfg.Sources))
	}

	// Local entries win on duplicate names.
	for _, s := range cfg.Sources {
		if s.Name == "overridden" && s.URL != "https://example.com/new.json" {
			t.Errorf("expected local override to win, got %s", s.URL)
		}
	}
}

func TestResolvedRegistryURL(t *testing.T) {
	cfg := &Config{}
	want := "https://raw.githubusercontent.com/saurabhsharma2u/iambot/main/registry/manifest.yaml"
	if got := cfg.ResolvedRegistryURL(); got != want {
		t.Errorf("expected %q, got %q", want, got)
	}

	cfg = &Config{RegistryRef: "v1.0.0"}
	want = "https://raw.githubusercontent.com/saurabhsharma2u/iambot/v1.0.0/registry/manifest.yaml"
	if got := cfg.ResolvedRegistryURL(); got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestMergeSources(t *testing.T) {
	base := []SourceConfig{
		{Name: "a", URL: "https://example.com/a.json"},
		{Name: "b", URL: "https://example.com/old.json"},
	}
	overlay := []SourceConfig{
		{Name: "b", URL: "https://example.com/new.json"},
		{Name: "c", URL: "https://example.com/c.json"},
	}

	merged := MergeSources(base, overlay)
	if len(merged) != 3 {
		t.Fatalf("expected 3 sources, got %d", len(merged))
	}
	if merged[0].Name != "a" || merged[1].Name != "b" || merged[2].Name != "c" {
		t.Errorf("unexpected order: %v", merged)
	}
	if merged[1].URL != "https://example.com/new.json" {
		t.Errorf("expected overlay to win, got %s", merged[1].URL)
	}
}

func TestLoadValidation(t *testing.T) {
	cases := map[string]string{
		"empty name":     "sources:\n  - name: \"\"\n    category: x\n    type: http\n    url: https://example.com/a.json\n    enabled: true\n",
		"unknown type":   "sources:\n  - name: a\n    category: x\n    type: ftp\n    url: https://example.com/a.json\n    enabled: true\n",
		"http w/o url":   "sources:\n  - name: a\n    category: x\n    type: http\n    enabled: true\n",
		"bad url":        "sources:\n  - name: a\n    category: x\n    type: http\n    url: \"::not-a-url\"\n    enabled: true\n",
		"file w/o path":  "sources:\n  - name: a\n    category: x\n    type: file\n    enabled: true\n",
		"duplicate name": "sources:\n  - name: a\n    category: x\n    type: http\n    url: https://example.com/a.json\n    enabled: true\n  - name: a\n    category: y\n    type: http\n    url: https://example.com/b.json\n    enabled: true\n",
	}

	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			path := t.TempDir() + "/botcheck.yaml"
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				t.Fatal(err)
			}
			if _, err := Load(path); err == nil {
				t.Errorf("expected validation error, got nil")
			}
		})
	}
}
