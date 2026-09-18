package registry

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestVerifyMapRoundTrip(t *testing.T) {
	dir := t.TempDir()
	in := map[string][]string{
		"yandex": {"yandex.ru", "yandex.net", "yandex.com"},
	}
	if err := SaveVerifyMap(dir, in); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := LoadVerifyMap(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(got["yandex"]) != 3 || got["yandex"][0] != "yandex.ru" {
		t.Errorf("unexpected map: %v", got)
	}
}

func TestVerifyMapMissing(t *testing.T) {
	if _, err := LoadVerifyMap(filepath.Join(t.TempDir(), "missing")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected ErrNotExist, got %v", err)
	}
}
