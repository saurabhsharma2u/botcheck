package source

import (
	"bufio"
	"context"
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/saurabhsharma2u/botcheck/internal/matcher"
)

type fileSource struct {
	name     string
	category string
	path     string
}

func NewFile(name, category, path string) Source {
	return &fileSource{name: name, category: category, path: path}
}

func (s *fileSource) Name() string {
	return s.name
}

func (s *fileSource) Category() string {
	return s.category
}

func (s *fileSource) Fetch(_ context.Context) ([]netip.Prefix, matcher.Meta, error) {
	f, err := os.Open(expandPath(s.path))
	if err != nil {
		return nil, matcher.Meta{}, fmt.Errorf("open file: %w", err)
	}
	defer func() { _ = f.Close() }()

	var prefixes []netip.Prefix
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 8*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
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
	if err := scanner.Err(); err != nil {
		return nil, matcher.Meta{}, fmt.Errorf("read file: %w", err)
	}

	if len(prefixes) == 0 {
		return nil, matcher.Meta{}, fmt.Errorf("no prefixes found in %s", s.path)
	}

	meta := matcher.Meta{
		Source:    s.name,
		Category:  s.category,
		Name:      s.name,
		UpdatedAt: time.Now(),
	}

	return prefixes, meta, nil
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			path = filepath.Join(home, path[2:])
		}
	}
	return os.ExpandEnv(path)
}
