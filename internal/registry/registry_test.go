package registry

import (
	"context"
	"errors"
	"net/netip"
	"path/filepath"
	"testing"

	"github.com/saurabhsharma2u/botcheck/internal/matcher"
)

func TestDiskRegistryEmpty(t *testing.T) {
	r := NewDiskRegistry(filepath.Join(t.TempDir(), "missing"))
	if err := r.Load(context.Background()); !errors.Is(err, ErrEmpty) {
		t.Fatalf("expected ErrEmpty, got %v", err)
	}
}

func TestDiskRegistry(t *testing.T) {
	tmp := t.TempDir()
	cacheDir := filepath.Join(tmp, "botcheck-test")

	r := NewDiskRegistry(cacheDir)
	if err := r.Load(context.Background()); !errors.Is(err, ErrEmpty) {
		t.Fatalf("expected ErrEmpty on fresh dir, got %v", err)
	}

	entries := []Entry{
		{
			Prefix: "192.168.1.0/24",
			Meta: matcher.Meta{
				Source: "test",
			},
		},
	}
	stats := Stats{
		TotalPrefixes: 1,
		Sources:       map[string]int{"test": 1},
	}

	err := r.SaveRaw(context.Background(), entries, stats)
	if err != nil {
		t.Fatal(err)
	}

	// Create new registry to verify persistence
	r2 := NewDiskRegistry(cacheDir)
	err = r2.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	ip := netip.MustParseAddr("192.168.1.5")
	hit, ok := r2.Contains(ip)
	if !ok {
		t.Fatal("expected hit")
	}
	if hit.Meta.Source != "test" {
		t.Fatalf("expected test, got %s", hit.Meta.Source)
	}
	if hit.Prefix.String() != "192.168.1.0/24" {
		t.Fatalf("expected prefix 192.168.1.0/24, got %s", hit.Prefix)
	}

	s := r2.Stats()
	if s.TotalPrefixes != 1 {
		t.Fatalf("expected 1, got %d", s.TotalPrefixes)
	}
	if s.Sources["test"] != 1 {
		t.Fatalf("expected 1, got %d", s.Sources["test"])
	}
}
