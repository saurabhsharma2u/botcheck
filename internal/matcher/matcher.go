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
	Lookup(ip netip.Addr) (Meta, bool)
	Len() int
}

type bartMatcher struct {
	trie *bart.Table[Meta]
	len  int
}

func New() Matcher {
	return &bartMatcher{
		trie: new(bart.Table[Meta]),
	}
}

func (m *bartMatcher) Insert(prefix netip.Prefix, meta Meta) {
	m.trie.Insert(prefix, meta)
	m.len++
}

func (m *bartMatcher) Lookup(ip netip.Addr) (Meta, bool) {
	meta, ok := m.trie.Lookup(ip)
	return meta, ok
}

func (m *bartMatcher) Len() int {
	return m.len
}
