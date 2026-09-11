package config

import (
	"fmt"
	"os"

	yaml "gopkg.in/yaml.v3"
)

type Config struct {
	CacheDir string         `yaml:"cache_dir"`
	Sources  []SourceConfig `yaml:"sources"`
}

type SourceConfig struct {
	Name     string `yaml:"name"`
	Category string `yaml:"category"`
	Type     string `yaml:"type"`
	URL      string `yaml:"url"`
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
	return &cfg, nil
}
