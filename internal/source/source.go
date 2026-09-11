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
