package matcher

import (
	"net/netip"
	"time"

	"github.com/gaissmai/bart"
)

type Meta struct {
	Source    string            `json:"source"`
	Category  string            `json:"category"`
	Name      string            `json:"name"`
	UpdatedAt time.Time         `json:"updated_at"`
	Extra     map[string]string `json:"extra,omitempty"`
}

type Hit struct {
	Prefix netip.Prefix
	Meta   Meta
}

type Matcher interface {
	Insert(prefix netip.Prefix, meta Meta)
	Lookup(ip netip.Addr) (netip.Prefix, Meta, bool)
	Len() int
}

type bartMatcher struct {
	trie *bart.Table[Meta]
}

func New() Matcher {
	return &bartMatcher{
		trie: new(bart.Table[Meta]),
	}
}

func (m *bartMatcher) Insert(prefix netip.Prefix, meta Meta) {
	m.trie.Insert(prefix, meta)
}

func (m *bartMatcher) Lookup(ip netip.Addr) (netip.Prefix, Meta, bool) {
	if !ip.IsValid() {
		return netip.Prefix{}, Meta{}, false
	}
	ip = ip.WithZone("").Unmap()
	bits := 128
	if ip.Is4() {
		bits = 32
	}
	return m.trie.LookupPrefixLPM(netip.PrefixFrom(ip, bits))
}

func (m *bartMatcher) Len() int {
	return m.trie.Size()
}
