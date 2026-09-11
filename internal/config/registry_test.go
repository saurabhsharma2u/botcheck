package config

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestFetchRegistryManifestFile(t *testing.T) {
	content := []byte("version: \"2026-09-11\"\nsources:\n  - name: googlebot\n    category: search\n    type: http\n    url: https://example.com/googlebot.json\n    enabled: true\n")
	path := filepath.Join(t.TempDir(), "manifest.yaml")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	man, err := FetchRegistryManifest(context.Background(), path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if man.Version != "2026-09-11" {
		t.Errorf("expected version 2026-09-11, got %q", man.Version)
	}
	if len(man.Sources) != 1 || man.Sources[0].Name != "googlebot" {
		t.Errorf("unexpected sources: %v", man.Sources)
	}
}

func TestFetchRegistryManifestHTTP(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/yaml")
		_, _ = w.Write([]byte("version: \"v1\"\nsources: []\n"))
	}))
	defer ts.Close()

	man, err := FetchRegistryManifest(context.Background(), ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if man.Version != "v1" {
		t.Errorf("expected version v1, got %q", man.Version)
	}
}

func TestFetchRegistryManifestHTTPError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	if _, err := FetchRegistryManifest(context.Background(), ts.URL); err == nil {
		t.Error("expected error for 404, got nil")
	}
}
