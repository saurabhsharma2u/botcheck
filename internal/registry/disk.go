package registry

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/saurabhsharma2u/botcheck/internal/matcher"
)

type diskRegistry struct {
	cacheDir string
	mu       sync.RWMutex
	m        matcher.Matcher
	stats    Stats
}

func NewDiskRegistry(cacheDir string) Registry {
	if cacheDir == "" {
		if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
			cacheDir = filepath.Join(xdg, "botcheck")
		} else {
			home, _ := os.UserHomeDir()
			cacheDir = filepath.Join(home, ".cache", "botcheck")
		}
	}
	return &diskRegistry{
		cacheDir: cacheDir,
		m:        matcher.New(),
	}
}

func (r *diskRegistry) manifestPath() string {
	return filepath.Join(r.cacheDir, "manifest.json")
}

func (r *diskRegistry) dataPath() string {
	return filepath.Join(r.cacheDir, "data", "prefixes.json.gz")
}

func (r *diskRegistry) Load(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	mfPath := r.manifestPath()
	b, err := os.ReadFile(mfPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ErrEmpty
		}
		return fmt.Errorf("read manifest: %w", err)
	}

	var man Manifest
	if err := json.Unmarshal(b, &man); err != nil {
		return fmt.Errorf("parse manifest: %w", err)
	}

	r.stats = man.Stats
	r.stats.LastUpdated = man.UpdatedAt

	dPath := r.dataPath()
	f, err := os.Open(dPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ErrEmpty
		}
		return fmt.Errorf("open data: %w", err)
	}
	defer func() { _ = f.Close() }()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("gzip reader: %w", err)
	}
	defer func() { _ = gr.Close() }()

	dec := json.NewDecoder(gr)
	var entries []Entry
	if err := dec.Decode(&entries); err != nil {
		return fmt.Errorf("decode data: %w", err)
	}

	newM := matcher.New()
	for _, e := range entries {
		if prefix, err := netip.ParsePrefix(e.Prefix); err == nil {
			newM.Insert(prefix, e.Meta)
		}
	}
	if newM.Len() == 0 {
		return ErrEmpty
	}
	r.m = newM

	return nil
}

func (r *diskRegistry) SaveRaw(ctx context.Context, entries []Entry, stats Stats) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	dataDir := filepath.Join(r.cacheDir, "data")
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	if err := writeAtomic(r.dataPath(), 0600, func(f *os.File) error {
		gw := gzip.NewWriter(f)
		enc := json.NewEncoder(gw)
		if err := enc.Encode(entries); err != nil {
			_ = gw.Close()
			return err
		}
		return gw.Close()
	}); err != nil {
		return fmt.Errorf("save data: %w", err)
	}

	man := Manifest{
		Version:   1,
		UpdatedAt: time.Now(),
		Stats:     stats,
	}
	b, err := json.MarshalIndent(man, "", "  ")
	if err != nil {
		return err
	}
	if err := writeAtomic(r.manifestPath(), 0600, func(f *os.File) error {
		_, err := f.Write(b)
		return err
	}); err != nil {
		return fmt.Errorf("save manifest: %w", err)
	}

	r.stats = stats
	r.stats.LastUpdated = man.UpdatedAt

	newM := matcher.New()
	for _, e := range entries {
		if prefix, err := netip.ParsePrefix(e.Prefix); err == nil {
			newM.Insert(prefix, e.Meta)
		}
	}
	r.m = newM

	return nil
}

func writeAtomic(path string, perm os.FileMode, write func(f *os.File) error) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if err := write(tmp); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, perm); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func (r *diskRegistry) Contains(ip netip.Addr) (matcher.Hit, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	prefix, meta, ok := r.m.Lookup(ip)
	if !ok {
		return matcher.Hit{}, false
	}
	return matcher.Hit{Prefix: prefix, Meta: meta}, true
}

func (r *diskRegistry) Stats() Stats {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.stats
}

func (r *diskRegistry) LastUpdated() time.Time {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.stats.LastUpdated
}
