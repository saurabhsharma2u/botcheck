package dnsverify

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"strings"
	"testing"
)

type fakeResolver struct {
	ptrs  map[string][]string
	fwd   map[string][]string
	err   map[string]error
	calls int
}

func (f *fakeResolver) LookupAddr(_ context.Context, addr string) ([]string, error) {
	f.calls++
	if err, ok := f.err["ptr:"+addr]; ok {
		return nil, err
	}
	return f.ptrs[addr], nil
}

func (f *fakeResolver) LookupIPAddr(_ context.Context, host string) ([]net.IPAddr, error) {
	f.calls++
	if err, ok := f.err["fwd:"+host]; ok {
		return nil, err
	}
	var out []net.IPAddr
	for _, s := range f.fwd[host] {
		out = append(out, net.IPAddr{IP: net.ParseIP(s)})
	}
	return out, nil
}

func TestVerifyOK(t *testing.T) {
	r := &fakeResolver{
		ptrs: map[string][]string{"66.249.66.1": {"crawl-66-249-66-1.googlebot.com."}},
		fwd:  map[string][]string{"crawl-66-249-66-1.googlebot.com": {"66.249.66.1"}},
	}
	vd, err := Verify(context.Background(), r, netip.MustParseAddr("66.249.66.1"), []string{"googlebot.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !vd.Verified || vd.Hostname != "crawl-66-249-66-1.googlebot.com" {
		t.Errorf("unexpected verdict: %+v", vd)
	}
}

func TestVerifyEvilSuffix(t *testing.T) {
	r := &fakeResolver{
		ptrs: map[string][]string{"1.2.3.4": {"crawl-1-2-3-4.evil-googlebot.com."}},
		fwd:  map[string][]string{"crawl-1-2-3-4.evil-googlebot.com": {"1.2.3.4"}},
	}
	vd, err := Verify(context.Background(), r, netip.MustParseAddr("1.2.3.4"), []string{"googlebot.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vd.Verified || !strings.Contains(vd.Reason, "suffix mismatch") {
		t.Errorf("evil suffix must not verify: %+v", vd)
	}
}

func TestVerifyForwardMismatch(t *testing.T) {
	r := &fakeResolver{
		ptrs: map[string][]string{"1.2.3.4": {"fake.googlebot.com."}},
		fwd:  map[string][]string{"fake.googlebot.com": {"9.9.9.9"}},
	}
	vd, err := Verify(context.Background(), r, netip.MustParseAddr("1.2.3.4"), []string{"googlebot.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vd.Verified || !strings.Contains(vd.Reason, "forward confirmation") {
		t.Errorf("forward mismatch must not verify: %+v", vd)
	}
}

func TestVerifySecondPTRVerifies(t *testing.T) {
	r := &fakeResolver{
		ptrs: map[string][]string{"1.2.3.4": {"junk.example.com.", "real.googlebot.com."}},
		fwd: map[string][]string{
			"junk.example.com":   {"1.2.3.4"},
			"real.googlebot.com": {"1.2.3.4"},
		},
	}
	vd, err := Verify(context.Background(), r, netip.MustParseAddr("1.2.3.4"), []string{"googlebot.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !vd.Verified || vd.Hostname != "real.googlebot.com" {
		t.Errorf("second PTR should verify: %+v", vd)
	}
}

func TestVerifyNoSuffixesUnverifiable(t *testing.T) {
	r := &fakeResolver{}
	vd, err := Verify(context.Background(), r, netip.MustParseAddr("1.2.3.4"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vd.Verified || !strings.Contains(vd.Reason, "unverifiable") {
		t.Errorf("empty suffixes must be unverifiable, not spoof: %+v", vd)
	}
	if r.calls != 0 {
		t.Errorf("no DNS lookups expected without suffixes, got %d", r.calls)
	}
}

func TestVerifyDNSError(t *testing.T) {
	r := &fakeResolver{err: map[string]error{"ptr:1.2.3.4": errors.New("timeout")}}
	if _, err := Verify(context.Background(), r, netip.MustParseAddr("1.2.3.4"), []string{"googlebot.com"}); err == nil {
		t.Error("expected DNS error, got nil")
	}
}

func TestVerifierCaches(t *testing.T) {
	r := &fakeResolver{
		ptrs: map[string][]string{"1.2.3.4": {"h.googlebot.com."}},
		fwd:  map[string][]string{"h.googlebot.com": {"1.2.3.4"}},
	}
	v := NewVerifierWithResolver(r)
	ip := netip.MustParseAddr("1.2.3.4")
	if _, err := v.VerifyIP(context.Background(), ip, []string{"googlebot.com"}); err != nil {
		t.Fatal(err)
	}
	if _, err := v.VerifyIP(context.Background(), ip, []string{"googlebot.com"}); err != nil {
		t.Fatal(err)
	}
	if r.calls != 2 {
		t.Errorf("expected 2 resolver calls (one verify, one cached), got %d", r.calls)
	}
}

func TestSuffixesParsing(t *testing.T) {
	got := Suffixes(map[string]string{ExtraKey: " yandex.ru, YANDEX.NET ,,yandex.com "})
	want := []string{"yandex.ru", "yandex.net", "yandex.com"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
	if Suffixes(nil) != nil {
		t.Error("nil extra must yield nil suffixes")
	}
}

func TestVerifyAnyOK(t *testing.T) {
	r := &fakeResolver{
		ptrs: map[string][]string{"95.108.1.2": {"spider-95-108-1-2.yandex.com."}},
		fwd:  map[string][]string{"spider-95-108-1-2.yandex.com": {"95.108.1.2"}},
	}
	bySource := map[string][]string{
		"google-common-crawlers": {"googlebot.com"},
		"yandex":                 {"yandex.ru", "yandex.net", "yandex.com"},
	}
	owners, vd, ptrs, err := VerifyAny(context.Background(), r, netip.MustParseAddr("95.108.1.2"), bySource)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !vd.Verified || vd.Hostname != "spider-95-108-1-2.yandex.com" {
		t.Errorf("unexpected verdict: %+v", vd)
	}
	if len(owners) != 1 || owners[0] != "yandex" {
		t.Errorf("unexpected owners: %v", owners)
	}
	if len(ptrs) != 1 {
		t.Errorf("expected PTR hint, got %v", ptrs)
	}
}

func TestVerifyAnyEvilSuffix(t *testing.T) {
	r := &fakeResolver{
		ptrs: map[string][]string{"1.2.3.4": {"x.evil-yandex.com."}},
		fwd:  map[string][]string{"x.evil-yandex.com": {"1.2.3.4"}},
	}
	bySource := map[string][]string{"yandex": {"yandex.com"}}
	owners, vd, ptrs, err := VerifyAny(context.Background(), r, netip.MustParseAddr("1.2.3.4"), bySource)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vd.Verified || len(owners) != 0 {
		t.Errorf("evil suffix must not verify: %+v %v", vd, owners)
	}
	if len(ptrs) != 1 {
		t.Errorf("expected PTR hint even on mismatch, got %v", ptrs)
	}
}

func TestVerifyAnyForwardMismatchAttributesOwner(t *testing.T) {
	r := &fakeResolver{
		ptrs: map[string][]string{"1.2.3.4": {"fake.googlebot.com."}},
		fwd:  map[string][]string{"fake.googlebot.com": {"9.9.9.9"}},
	}
	bySource := map[string][]string{"google-common-crawlers": {"googlebot.com"}}
	owners, vd, _, err := VerifyAny(context.Background(), r, netip.MustParseAddr("1.2.3.4"), bySource)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vd.Verified || !strings.Contains(vd.Reason, "forward confirmation") {
		t.Errorf("unexpected verdict: %+v", vd)
	}
	if len(owners) != 1 || owners[0] != "google-common-crawlers" {
		t.Errorf("failure should attribute the claiming source, got %v", owners)
	}
}

func TestVerifyAnyNoPTRIsSoft(t *testing.T) {
	r := &fakeResolver{err: map[string]error{
		"ptr:1.2.3.4": &net.DNSError{Err: "no such host", IsNotFound: true},
	}}
	bySource := map[string][]string{"yandex": {"yandex.com"}}
	_, vd, _, err := VerifyAny(context.Background(), r, netip.MustParseAddr("1.2.3.4"), bySource)
	if err != nil {
		t.Fatalf("NXDOMAIN must be soft, got error: %v", err)
	}
	if vd.Verified || !strings.Contains(vd.Reason, "no PTR") {
		t.Errorf("unexpected verdict: %+v", vd)
	}
}

func TestVerifyAnyTimeoutIsHard(t *testing.T) {
	r := &fakeResolver{err: map[string]error{
		"ptr:1.2.3.4": &net.DNSError{Err: "timeout", IsTimeout: true},
	}}
	bySource := map[string][]string{"yandex": {"yandex.com"}}
	if _, _, _, err := VerifyAny(context.Background(), r, netip.MustParseAddr("1.2.3.4"), bySource); err == nil {
		t.Error("timeout must be a hard error, got nil")
	}
}

func TestVerifyAnyEmptyMapNoLookups(t *testing.T) {
	r := &fakeResolver{}
	_, vd, _, err := VerifyAny(context.Background(), r, netip.MustParseAddr("1.2.3.4"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vd.Verified || !strings.Contains(vd.Reason, "unverifiable") {
		t.Errorf("unexpected verdict: %+v", vd)
	}
	if r.calls != 0 {
		t.Errorf("no DNS lookups expected, got %d", r.calls)
	}
}

func TestVerifyNotFoundIsSoft(t *testing.T) {
	r := &fakeResolver{err: map[string]error{
		"ptr:1.2.3.4": &net.DNSError{Err: "no such host", IsNotFound: true},
	}}
	vd, err := Verify(context.Background(), r, netip.MustParseAddr("1.2.3.4"), []string{"googlebot.com"})
	if err != nil {
		t.Fatalf("NXDOMAIN must be soft, got error: %v", err)
	}
	if vd.Verified || !strings.Contains(vd.Reason, "no PTR") {
		t.Errorf("unexpected verdict: %+v", vd)
	}
}
