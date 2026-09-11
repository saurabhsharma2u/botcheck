package source

import (
	"context"
	"net/netip"

	"github.com/saurabhsharma2u/iambot/internal/matcher"
)

type Source interface {
	Name() string
	Category() string
	Fetch(ctx context.Context) ([]netip.Prefix, matcher.Meta, error)
}

func parsePrefixOrAddr(s string) (netip.Prefix, bool) {
	if parsed, err := netip.ParsePrefix(s); err == nil {
		return parsed, true
	}
	if addr, err := netip.ParseAddr(s); err == nil {
		bits := 128
		if addr.Is4() {
			bits = 32
		}
		return netip.PrefixFrom(addr, bits), true
	}
	return netip.Prefix{}, false
}
