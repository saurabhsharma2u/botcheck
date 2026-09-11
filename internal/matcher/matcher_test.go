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

	if m.Len() != 1 {
		t.Errorf("expected len 1, got %d", m.Len())
	}

	ip := netip.MustParseAddr("192.168.1.5")
	hitMeta, ok := m.Lookup(ip)
	if !ok {
		t.Errorf("expected hit for %s", ip)
	}
	if hitMeta.Source != "test" {
		t.Errorf("expected source test, got %s", hitMeta.Source)
	}

	ip2 := netip.MustParseAddr("10.0.0.1")
	_, ok = m.Lookup(ip2)
	if ok {
		t.Errorf("expected no hit for %s", ip2)
	}
}
