// Package dnsverify implements forward-confirmed reverse DNS (FcRDNS)
// checks: PTR the IP, require the hostname to end at one of the source's
// documented suffixes, then resolve the hostname forward and require it
// to map back to the same IP. Either direction skipped is spoofable, so
// both are mandatory.
package dnsverify

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"sort"
	"strings"
	"sync"
)

// ExtraKey is the matcher.Meta.Extra key under which refresh stores the
// comma-joined verification suffixes for a source.
const ExtraKey = "verify_suffixes"

// Resolver abstracts DNS lookups so tests never touch the network.
// *net.Resolver implements it directly.
type Resolver interface {
	LookupAddr(ctx context.Context, addr string) ([]string, error)
	LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error)
}

// Verdict is the outcome of verifying one IP.
type Verdict struct {
	// Verified is true only if a PTR hostname matched a suffix AND
	// forward-resolved back to the IP.
	Verified bool
	// Hostname is the PTR name that verified (empty unless Verified).
	Hostname string
	// Reason explains Verified=false: "unverifiable" (no known suffixes),
	// "suffix mismatch", or "forward confirmation failed".
	Reason string
}

// Suffixes extracts verification suffixes from cached source metadata.
func Suffixes(extra map[string]string) []string {
	raw, ok := extra[ExtraKey]
	if !ok {
		return nil
	}
	var out []string
	for _, s := range strings.Split(raw, ",") {
		if s = strings.ToLower(strings.TrimSpace(s)); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// matchSuffix reports whether host equals or sits under suffix.
// Dot-boundary matching rejects evil-googlebot.com for googlebot.com.
func matchSuffix(host, suffix string) bool {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	suffix = strings.ToLower(strings.TrimSpace(suffix))
	return host == suffix || strings.HasSuffix(host, "."+suffix)
}

// isNotFound reports NXDOMAIN-style answers (no PTR record), which are
// definitive, unlike timeouts/refusals which are transient failures.
func isNotFound(err error) bool {
	var dnsErr *net.DNSError
	return errors.As(err, &dnsErr) && dnsErr.IsNotFound
}

// forwardConfirms resolves host and reports whether it maps back to ip.
func forwardConfirms(ctx context.Context, r Resolver, host string, ip netip.Addr) bool {
	addrs, err := r.LookupIPAddr(ctx, host)
	if err != nil {
		return false
	}
	for _, a := range addrs {
		if addr, ok := netip.AddrFromSlice(a.IP); ok && addr.Unmap() == ip.Unmap() {
			return true
		}
	}
	return false
}

// Verify runs FcRDNS for ip against suffixes. A nil/empty suffix list
// yields an "unverifiable" verdict, never a spoof verdict.
func Verify(ctx context.Context, r Resolver, ip netip.Addr, suffixes []string) (Verdict, error) {
	if len(suffixes) == 0 {
		return Verdict{Reason: "unverifiable: no known verification domains for this source"}, nil
	}

	ptrs, err := r.LookupAddr(ctx, ip.String())
	if err != nil {
		if isNotFound(err) {
			return Verdict{Reason: "no PTR record for IP"}, nil
		}
		return Verdict{}, fmt.Errorf("reverse lookup: %w", err)
	}

	suffixMatched := false
	for _, ptr := range ptrs {
		host := strings.TrimSuffix(ptr, ".")
		matched := false
		for _, s := range suffixes {
			if matchSuffix(host, s) {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}
		suffixMatched = true

		if forwardConfirms(ctx, r, host, ip) {
			return Verdict{Verified: true, Hostname: host}, nil
		}
	}

	if !suffixMatched {
		return Verdict{Reason: "suffix mismatch: PTR hostname is outside known verification domains"}, nil
	}
	return Verdict{Reason: "forward confirmation failed: hostname does not resolve back to this IP"}, nil
}

// VerifyAny checks ip against every known source's suffixes, for IPs with
// no registry match. It returns the owning source(s), the verdict, and the
// raw PTR hostnames (useful as a triage hint when nothing verifies).
//
// Loop safety: strictly single-pass. The PTR list is deduplicated, each
// distinct hostname is forward-resolved at most once, and forward results
// (bare IPs) never re-enter reverse lookup — no recursion, no retries.
// An empty map yields "unverifiable" with zero DNS lookups.
func VerifyAny(ctx context.Context, r Resolver, ip netip.Addr, bySource map[string][]string) ([]string, Verdict, []string, error) {
	if len(bySource) == 0 {
		return nil, Verdict{Reason: "unverifiable: no verification domains known (run refresh)"}, nil, nil
	}

	raw, err := r.LookupAddr(ctx, ip.String())
	if err != nil {
		if isNotFound(err) {
			return nil, Verdict{Reason: "no PTR record for IP"}, nil, nil
		}
		return nil, Verdict{}, nil, fmt.Errorf("reverse lookup: %w", err)
	}

	// Deduplicate normalized hostnames, preserving resolver order.
	var ptrs []string
	seen := make(map[string]bool, len(raw))
	for _, p := range raw {
		p = strings.TrimSuffix(p, ".")
		if !seen[strings.ToLower(p)] {
			seen[strings.ToLower(p)] = true
			ptrs = append(ptrs, p)
		}
	}

	ownersOf := func(host string) []string {
		var owners []string
		for src, suffixes := range bySource {
			for _, s := range suffixes {
				if matchSuffix(host, s) {
					owners = append(owners, src)
					break
				}
			}
		}
		sort.Strings(owners)
		return owners
	}

	var failedOwners []string
	for _, ptr := range ptrs {
		owners := ownersOf(ptr)
		if len(owners) == 0 {
			continue
		}
		if forwardConfirms(ctx, r, ptr, ip) {
			return owners, Verdict{Verified: true, Hostname: ptr}, ptrs, nil
		}
		if failedOwners == nil {
			failedOwners = owners
		}
	}

	if failedOwners != nil {
		return failedOwners, Verdict{Reason: "forward confirmation failed: hostname does not resolve back to this IP"}, ptrs, nil
	}
	return nil, Verdict{Reason: "suffix mismatch: PTR hostname is outside all known verification domains"}, ptrs, nil
}

// Verifier caches verdicts by IP so streaming commands (scan, a future
// tail/watch mode) pay one lookup per unique IP instead of per line.
type Verifier struct {
	r Resolver

	mu    sync.Mutex
	cache map[string]Verdict
}

// NewVerifier returns a Verifier using the system resolver.
func NewVerifier() *Verifier {
	return NewVerifierWithResolver(&net.Resolver{})
}

// NewVerifierWithResolver returns a Verifier using r (tests inject a fake).
func NewVerifierWithResolver(r Resolver) *Verifier {
	return &Verifier{r: r, cache: make(map[string]Verdict)}
}

// VerifyIP returns the cached verdict for ip when present.
func (v *Verifier) VerifyIP(ctx context.Context, ip netip.Addr, suffixes []string) (Verdict, error) {
	key := ip.String() + "\x00" + strings.Join(suffixes, ",")
	v.mu.Lock()
	if vd, ok := v.cache[key]; ok {
		v.mu.Unlock()
		return vd, nil
	}
	v.mu.Unlock()

	vd, err := Verify(ctx, v.r, ip, suffixes)
	if err != nil {
		return vd, err
	}
	v.mu.Lock()
	v.cache[key] = vd
	v.mu.Unlock()
	return vd, nil
}
