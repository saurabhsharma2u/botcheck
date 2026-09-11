package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

const (
	DefaultRegistryURL = "https://raw.githubusercontent.com/saurabhsharma2u/iambot/{ref}/registry/manifest.yaml"
	DefaultRegistryRef = "main"
)

type Config struct {
	CacheDir    string         `yaml:"cache_dir"`
	RegistryURL string         `yaml:"registry_url"`
	RegistryRef string         `yaml:"registry_ref"`
	Imports     []string       `yaml:"imports"`
	Sources     []SourceConfig `yaml:"sources"`
}

type SourceConfig struct {
	Name     string `yaml:"name"`
	Category string `yaml:"category"`
	Type     string `yaml:"type"`
	URL      string `yaml:"url"`
	Path     string `yaml:"path"`
	Enabled  bool   `yaml:"enabled"`
}

func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, fmt.Errorf("parse config file: %w", err)
	}
	if err := cfg.validate(path); err != nil {
		return nil, err
	}
	if err := cfg.resolveImports(path, make(map[string]bool)); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// ResolvedRegistryURL returns the manifest location with {ref} substituted.
func (c *Config) ResolvedRegistryURL() string {
	rawURL := c.RegistryURL
	if rawURL == "" {
		rawURL = DefaultRegistryURL
	}
	ref := c.RegistryRef
	if ref == "" {
		ref = DefaultRegistryRef
	}
	return strings.ReplaceAll(rawURL, "{ref}", ref)
}

// MergeSources layers overlay on top of base; overlay wins on duplicate names.
func MergeSources(base, overlay []SourceConfig) []SourceConfig {
	return dedupeSources(append(base, overlay...))
}

func (c *Config) validate(origin string) error {
	seen := make(map[string]bool, len(c.Sources))
	for _, s := range c.Sources {
		if s.Name == "" {
			return fmt.Errorf("%s: source with empty name", origin)
		}
		if seen[s.Name] {
			return fmt.Errorf("%s: duplicate source %q", origin, s.Name)
		}
		seen[s.Name] = true
		switch s.Type {
		case "http":
			if s.URL == "" {
				return fmt.Errorf("%s: source %q: http requires url", origin, s.Name)
			}
			if _, err := url.ParseRequestURI(s.URL); err != nil {
				return fmt.Errorf("%s: source %q: invalid url: %w", origin, s.Name, err)
			}
		case "file":
			if s.Path == "" {
				return fmt.Errorf("%s: source %q: file requires path", origin, s.Name)
			}
		default:
			return fmt.Errorf("%s: source %q: unsupported type %q", origin, s.Name, s.Type)
		}
	}
	return nil
}

// resolveImports merges `imports:` files underneath this file's sources;
// local entries win on duplicate names.
func (c *Config) resolveImports(parent string, visited map[string]bool) error {
	if len(c.Imports) == 0 {
		return nil
	}
	dir := filepath.Dir(parent)
	var imported []SourceConfig
	for _, pattern := range c.Imports {
		if !filepath.IsAbs(pattern) {
			pattern = filepath.Join(dir, pattern)
		}
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return fmt.Errorf("resolve import %q: %w", pattern, err)
		}
		for _, m := range matches {
			if visited[m] {
				continue
			}
			visited[m] = true
			b, err := os.ReadFile(m)
			if err != nil {
				return fmt.Errorf("read import %q: %w", m, err)
			}
			var sub Config
			if err := yaml.Unmarshal(b, &sub); err != nil {
				return fmt.Errorf("parse import %q: %w", m, err)
			}
			if err := sub.validate(m); err != nil {
				return err
			}
			if err := sub.resolveImports(m, visited); err != nil {
				return err
			}
			imported = append(imported, sub.Sources...)
		}
	}
	c.Sources = MergeSources(imported, c.Sources)
	return nil
}

// dedupeSources drops duplicate source names, last occurrence wins,
// preserving first-seen order.
func dedupeSources(sources []SourceConfig) []SourceConfig {
	index := make(map[string]int, len(sources))
	out := make([]SourceConfig, 0, len(sources))
	for _, s := range sources {
		if i, ok := index[s.Name]; ok {
			out[i] = s
			continue
		}
		index[s.Name] = len(out)
		out = append(out, s)
	}
	return out
}
