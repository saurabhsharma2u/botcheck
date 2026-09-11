package source

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"time"

	"github.com/saurabhsharma2u/iambot/internal/matcher"
)

type httpSource struct {
	name     string
	category string
	url      string
	client   *http.Client
}

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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, matcher.Meta{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, matcher.Meta{}, fmt.Errorf("read response: %w", err)
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
				if parsed, err := netip.ParsePrefix(pStr); err == nil {
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
						if parsed, err := netip.ParsePrefix(strVal); err == nil {
							prefixes = append(prefixes, parsed)
						}
					}
				}
			}
		}
	}

	if len(prefixes) == 0 {
		return nil, matcher.Meta{}, fmt.Errorf("unsupported json format or no prefixes found for %s", s.url)
	}

	meta := matcher.Meta{
		Source:    s.name,
		Category:  s.category,
		Name:      s.name,
		UpdatedAt: time.Now(),
	}

	return prefixes, meta, nil
}
