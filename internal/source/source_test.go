package source

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHTTPFetch(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"prefixes": [
				{"ipv4Prefix": "192.168.0.0/16"},
				{"ipv6Prefix": "2001:db8::/32"}
			]
		}`))
	}))
	defer ts.Close()

	s := NewHTTP("test", "good", ts.URL)
	prefixes, meta, err := s.Fetch(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if meta.Source != "test" {
		t.Errorf("expected test, got %s", meta.Source)
	}

	if len(prefixes) != 2 {
		t.Errorf("expected 2 prefixes, got %d", len(prefixes))
	}
}

func TestHTTPFetchBareIPList(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"ips": [
				{"ip_address": "5.39.1.224"},
				{"ip_address": "2001:db8::1"}
			]
		}`))
	}))
	defer ts.Close()

	s := NewHTTP("ahrefs", "seo", ts.URL)
	prefixes, _, err := s.Fetch(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(prefixes) != 2 {
		t.Fatalf("expected 2 prefixes, got %d", len(prefixes))
	}

	if prefixes[0].String() != "5.39.1.224/32" {
		t.Errorf("expected 5.39.1.224/32, got %s", prefixes[0])
	}

	if prefixes[1].String() != "2001:db8::1/128" {
		t.Errorf("expected 2001:db8::1/128, got %s", prefixes[1])
	}
}

func TestHTTPFetchPlainText(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("# comment\n\n31.13.24.0/21\n157.240.0.35\nnot-an-ip\n"))
	}))
	defer ts.Close()

	s := NewHTTP("facebook", "social", ts.URL)
	prefixes, _, err := s.Fetch(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(prefixes) != 2 {
		t.Fatalf("expected 2 prefixes, got %d (%v)", len(prefixes), prefixes)
	}
	if prefixes[0].String() != "31.13.24.0/21" {
		t.Errorf("expected 31.13.24.0/21, got %s", prefixes[0])
	}
	if prefixes[1].String() != "157.240.0.35/32" {
		t.Errorf("expected 157.240.0.35/32, got %s", prefixes[1])
	}
}

func TestFacebookRegistryFile(t *testing.T) {
	content, err := os.ReadFile("../../registry/meta/fb.txt")
	if err != nil {
		t.Fatalf("read fb.txt: %v", err)
	}

	seen := make(map[string]bool)
	count := 0
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if _, ok := parsePrefixOrAddr(line); !ok {
			t.Errorf("unparseable line: %q", line)
		}
		if seen[line] {
			t.Errorf("duplicate line: %q", line)
		}
		seen[line] = true
		count++
	}

	if count < 1000 {
		t.Errorf("expected 1000+ prefixes in fb.txt, got %d", count)
	}

	for _, want := range []string{"31.13.24.0/21", "2a03:2880::/32", "57.144.0.0/14"} {
		if !seen[want] {
			t.Errorf("expected %q in fb.txt, missing", want)
		}
	}
	for _, absent := range []string{"157.240.4.0/24", "2a03:2881:1d::/48", "2a03:2880:f249::/48"} {
		if seen[absent] {
			t.Errorf("expected %q absent from fb.txt, present", absent)
		}
	}
}

func TestYandexRegistryFile(t *testing.T) {
	content, err := os.ReadFile("../../registry/meta/yandex.txt")
	if err != nil {
		t.Fatalf("read yandex.txt: %v", err)
	}

	seen := make(map[string]bool)
	count := 0
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if _, ok := parsePrefixOrAddr(line); !ok {
			t.Errorf("unparseable line: %q", line)
		}
		if seen[line] {
			t.Errorf("duplicate line: %q", line)
		}
		seen[line] = true
		count++
	}

	if count != 16 {
		t.Errorf("expected 16 prefixes in yandex.txt, got %d", count)
	}

	for _, want := range []string{"77.88.0.0/18", "95.108.128.0/17", "2a02:6b8::/29"} {
		if !seen[want] {
			t.Errorf("expected %q in yandex.txt, missing", want)
		}
	}
}

func TestTwitterRegistryFile(t *testing.T) {
	content, err := os.ReadFile("../../registry/meta/twitter.txt")
	if err != nil {
		t.Fatalf("read twitter.txt: %v", err)
	}

	seen := make(map[string]bool)
	count := 0
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if _, ok := parsePrefixOrAddr(line); !ok {
			t.Errorf("unparseable line: %q", line)
		}
		if seen[line] {
			t.Errorf("duplicate line: %q", line)
		}
		seen[line] = true
		count++
	}

	if count != 3 {
		t.Errorf("expected 3 prefixes in twitter.txt, got %d", count)
	}

	for _, want := range []string{"199.16.156.0/22", "199.59.148.0/22", "199.59.150.0/24"} {
		if !seen[want] {
			t.Errorf("expected %q in twitter.txt, missing", want)
		}
	}
}

func TestAnthropicAPIRegistryFile(t *testing.T) {
	content, err := os.ReadFile("../../registry/meta/anthropic-api.txt")
	if err != nil {
		t.Fatalf("read anthropic-api.txt: %v", err)
	}

	seen := make(map[string]bool)
	count := 0
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if _, ok := parsePrefixOrAddr(line); !ok {
			t.Errorf("unparseable line: %q", line)
		}
		if seen[line] {
			t.Errorf("duplicate line: %q", line)
		}
		seen[line] = true
		count++
	}

	if count != 1 {
		t.Errorf("expected 1 prefix in anthropic-api.txt, got %d", count)
	}

	for _, want := range []string{"160.79.104.0/21"} {
		if !seen[want] {
			t.Errorf("expected %q in anthropic-api.txt, missing", want)
		}
	}
	for _, absent := range []string{"34.162.46.92/32", "160.79.104.0/23", "2607:6bc0::/48"} {
		if seen[absent] {
			t.Errorf("expected %q absent from anthropic-api.txt, present", absent)
		}
	}
}

func TestHTTPFetchTooLarge(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(bytes.Repeat([]byte("x"), maxFeedBytes+10))
	}))
	defer ts.Close()

	s := NewHTTP("big", "test", ts.URL)
	if _, _, err := s.Fetch(context.Background()); err == nil {
		t.Error("expected error for oversize feed, got nil")
	}
}

func TestFileFetchEnvPath(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "ranges.txt")
	if err := os.WriteFile(real, []byte("10.9.0.0/16\n"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("BOTCHECK_TEST_DIR", dir)

	s := NewFile("env", "test", "$BOTCHECK_TEST_DIR/ranges.txt")
	prefixes, _, err := s.Fetch(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prefixes) != 1 || prefixes[0].String() != "10.9.0.0/16" {
		t.Errorf("unexpected prefixes: %v", prefixes)
	}
}

func TestFileFetch(t *testing.T) {
	content := `# Meta crawler ranges (maintained list)
69.63.176.0/20
66.220.144.0/20, 69.63.184.0/21
31.13.64.0 ; legacy block
157.240.0.35

# empty lines and junk are skipped
not-an-ip
`
	f, err := os.CreateTemp("", "botcheck-file-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(f.Name()) }()
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	s := NewFile("facebook", "social", f.Name())
	prefixes, meta, err := s.Fetch(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if meta.Source != "facebook" {
		t.Errorf("expected facebook, got %s", meta.Source)
	}

	want := map[string]bool{
		"69.63.176.0/20":  false,
		"66.220.144.0/20": false,
		"69.63.184.0/21":  false,
		"31.13.64.0/32":   false,
		"157.240.0.35/32": false,
	}
	if len(prefixes) != len(want) {
		t.Fatalf("expected %d prefixes, got %d (%v)", len(want), len(prefixes), prefixes)
	}
	for _, p := range prefixes {
		if _, ok := want[p.String()]; !ok {
			t.Errorf("unexpected prefix %s", p)
		}
	}
}
