package registry

import (
	"context"
	"net/netip"
	"time"

	"github.com/saurabhsharma2u/iambot/internal/matcher"
)

type Stats struct {
	TotalPrefixes int            `json:"total_prefixes"`
	Sources       map[string]int `json:"sources"`
	Categories    map[string]int `json:"categories"`
	DataVersion   string         `json:"data_version,omitempty"`
	LastUpdated   time.Time      `json:"last_updated"`
}

type Entry struct {
	Prefix string       `json:"prefix"`
	Meta   matcher.Meta `json:"meta"`
}

type Registry interface {
	Load(ctx context.Context) error
	SaveRaw(ctx context.Context, entries []Entry, stats Stats) error
	Contains(ip netip.Addr) (matcher.Hit, bool)
	Stats() Stats
	LastUpdated() time.Time
	Matcher() matcher.Matcher
}
