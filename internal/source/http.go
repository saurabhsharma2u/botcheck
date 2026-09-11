package source

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/saurabhsharma2u/iambot/internal/matcher"
)

type httpSource struct {
	name     string
	category string
	url      string
	client   *http.Client
}

const maxFeedBytes = 5 << 20

func NewHTTP(name, category, url string) Source {
	return &httpSource{
		name:     name,
		category: category,
		url:      url,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *httpSource) Name() string {
	return s.name
}

func (s *httpSource) Category() string {
	return s.category
}

func (s *httpSource) Fetch(ctx context.Context) ([]netip.Prefix, matcher.Meta, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.url, nil)
	if err != nil {
		return nil, matcher.Meta{}, fmt.Errorf("create request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, matcher.Meta{}, fmt.Errorf("do request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, matcher.Meta{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	b, err := io.ReadAll(io.LimitReader(resp.Body, maxFeedBytes+1))
	if err != nil {
		return nil, matcher.Meta{}, fmt.Errorf("read response: %w", err)
	}
	if len(b) > maxFeedBytes {
		return nil, matcher.Meta{}, fmt.Errorf("response exceeds %d bytes for %s", maxFeedBytes, s.url)
	}

	var prefixes []netip.Prefix

	// Try extracting Google bot format first.
	var gFormat struct {
		Prefixes []struct {
			IPv4Prefix string `json:"ipv4Prefix"`
			IPv6Prefix string `json:"ipv6Prefix"`
		} `json:"prefixes"`
	}
	if err := json.Unmarshal(b, &gFormat); err == nil && len(gFormat.Prefixes) > 0 {
		for _, p := range gFormat.Prefixes {
			var pStr string
			if p.IPv4Prefix != "" {
				pStr = p.IPv4Prefix
			} else if p.IPv6Prefix != "" {
				pStr = p.IPv6Prefix
			}
			if pStr != "" {
				if parsed, ok := parsePrefixOrAddr(pStr); ok {
					prefixes = append(prefixes, parsed)
				}
			}
		}
	} else {
		// Fallback: search for any string that looks like a prefix or IP in generic json
		var genericFormat struct {
			Prefixes []map[string]interface{} `json:"prefixes"`
		}
		if err := json.Unmarshal(b, &genericFormat); err == nil && len(genericFormat.Prefixes) > 0 {
			for _, item := range genericFormat.Prefixes {
				for _, val := range item {
					if strVal, ok := val.(string); ok {
						if parsed, ok := parsePrefixOrAddr(strVal); ok {
							prefixes = append(prefixes, parsed)
						}
					}
				}
			}
		}
	}

	// Bare-IP list format, e.g. Ahrefs: {"ips": [{"ip_address": "5.39.1.224"}, ...]}
	if len(prefixes) == 0 {
		var ipListFormat struct {
			IPs []map[string]interface{} `json:"ips"`
		}
		if err := json.Unmarshal(b, &ipListFormat); err == nil && len(ipListFormat.IPs) > 0 {
			for _, item := range ipListFormat.IPs {
				for _, val := range item {
					if strVal, ok := val.(string); ok {
						if parsed, ok := parsePrefixOrAddr(strVal); ok {
							prefixes = append(prefixes, parsed)
						}
					}
				}
			}
		}
	}

	// Plain-text fallback, one CIDR or IP per line (# and ; comments skipped).
	if len(prefixes) == 0 {
		for _, line := range strings.Split(string(b), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
				continue
			}
			if i := strings.IndexAny(line, "#;"); i != -1 {
				line = strings.TrimSpace(line[:i])
			}
			for _, field := range strings.FieldsFunc(line, func(r rune) bool {
				return r == ',' || r == ' ' || r == '\t'
			}) {
				if parsed, ok := parsePrefixOrAddr(strings.TrimSpace(field)); ok {
					prefixes = append(prefixes, parsed)
				}
			}
		}
	}

	if len(prefixes) == 0 {
		return nil, matcher.Meta{}, fmt.Errorf("unsupported format or no prefixes found for %s", s.url)
	}

	meta := matcher.Meta{
		Source:    s.name,
		Category:  s.category,
		Name:      s.name,
		UpdatedAt: time.Now(),
	}

	return prefixes, meta, nil
}
