package registry

import (
	"context"
	"net/netip"
	"path/filepath"
	"testing"

	"github.com/saurabhsharma2u/iambot/internal/matcher"
)

func TestDiskRegistry(t *testing.T) {
	tmp := t.TempDir()
	cacheDir := filepath.Join(tmp, "botcheck-test")

	r := NewDiskRegistry(cacheDir)
	err := r.Load(context.Background())
	if err != nil {
		t.Fatal(err)
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

	err = r.SaveRaw(context.Background(), entries, stats)
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

	s := r2.Stats()
	if s.TotalPrefixes != 1 {
		t.Fatalf("expected 1, got %d", s.TotalPrefixes)
	}
	if s.Sources["test"] != 1 {
		t.Fatalf("expected 1, got %d", s.Sources["test"])
	}
}
