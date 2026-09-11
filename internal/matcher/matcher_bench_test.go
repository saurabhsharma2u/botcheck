package matcher

import (
	"fmt"
	"net/netip"
	"testing"
	"time"
)

func benchMatcher(prefixes int) Matcher {
	m := New()
	meta := Meta{Source: "bench", Category: "bench", Name: "bench", UpdatedAt: time.Now()}
	for i := 0; i < prefixes; i++ {
		m.Insert(netip.MustParsePrefix(fmt.Sprintf("10.%d.%d.0/24", (i/256)%256, i%256)), meta)
	}
	m.Insert(netip.MustParsePrefix("40.77.167.0/24"), meta)
	return m
}

func BenchmarkLookupHit(b *testing.B) {
	m := benchMatcher(10000)
	ip := netip.MustParseAddr("40.77.167.61")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, ok := m.Lookup(ip); !ok {
			b.Fatal("expected hit")
		}
	}
}

func BenchmarkLookupMiss(b *testing.B) {
	m := benchMatcher(10000)
	ip := netip.MustParseAddr("8.8.8.8")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, ok := m.Lookup(ip); ok {
			b.Fatal("expected miss")
		}
	}
}
