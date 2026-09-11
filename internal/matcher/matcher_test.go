package matcher

import (
	"net/netip"
	"testing"
	"time"
)

func TestBartMatcher(t *testing.T) {
	m := New()
	meta := Meta{
		Source:    "test",
		Category:  "good",
		Name:      "test bot",
		UpdatedAt: time.Now(),
	}

	prefix := netip.MustParsePrefix("192.168.1.0/24")
	m.Insert(prefix, meta)

	// Re-inserting the same prefix must not grow the table.
	m.Insert(prefix, meta)

	if m.Len() != 1 {
		t.Errorf("expected len 1, got %d", m.Len())
	}

	ip := netip.MustParseAddr("192.168.1.5")
	hitPrefix, hitMeta, ok := m.Lookup(ip)
	if !ok {
		t.Fatalf("expected hit for %s", ip)
	}
	if hitMeta.Source != "test" {
		t.Errorf("expected source test, got %s", hitMeta.Source)
	}
	if hitPrefix != prefix {
		t.Errorf("expected prefix %s, got %s", prefix, hitPrefix)
	}

	// Longest match wins with overlaps.
	m.Insert(netip.MustParsePrefix("192.168.1.0/28"), meta)
	if pfx, _, ok := m.Lookup(ip); !ok || pfx.String() != "192.168.1.0/28" {
		t.Errorf("expected longest match 192.168.1.0/28, got %s (ok=%v)", pfx, ok)
	}
	if m.Len() != 2 {
		t.Errorf("expected len 2, got %d", m.Len())
	}

	ip2 := netip.MustParseAddr("10.0.0.1")
	_, _, ok = m.Lookup(ip2)
	if ok {
		t.Errorf("expected no hit for %s", ip2)
	}

	if _, _, ok := m.Lookup(netip.Addr{}); ok {
		t.Error("expected no hit for invalid addr")
	}

	m.Insert(netip.MustParsePrefix("fe80::/10"), meta)
	if _, _, ok := m.Lookup(netip.MustParseAddr("fe80::1%eth0")); !ok {
		t.Error("expected zone-scoped link-local to match fe80::/10")
	}
}
