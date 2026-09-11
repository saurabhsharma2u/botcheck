package config

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v3"
)

type RegistryManifest struct {
	Version string         `yaml:"version"`
	Sources []SourceConfig `yaml:"sources"`
}

func FetchRegistryManifest(ctx context.Context, rawURL string) (*RegistryManifest, error) {
	var b []byte
	if strings.HasPrefix(rawURL, "http://") || strings.HasPrefix(rawURL, "https://") {
		client := &http.Client{Timeout: 15 * time.Second}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			return nil, fmt.Errorf("create registry request: %w", err)
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("fetch registry: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("fetch registry: unexpected status %d", resp.StatusCode)
		}
		b, err = io.ReadAll(io.LimitReader(resp.Body, 4<<20))
		if err != nil {
			return nil, fmt.Errorf("read registry: %w", err)
		}
	} else {
		var err error
		b, err = os.ReadFile(rawURL)
		if err != nil {
			return nil, fmt.Errorf("read registry file: %w", err)
		}
	}

	var man RegistryManifest
	if err := yaml.Unmarshal(b, &man); err != nil {
		return nil, fmt.Errorf("parse registry manifest: %w", err)
	}
	return &man, nil
}
