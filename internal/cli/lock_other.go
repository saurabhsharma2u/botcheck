//go:build !darwin && !linux

package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Fallback for platforms without flock: exclusive-create a lock file.
// Unlike flock this can go stale on crash; the error message tells the
// operator how to clear it.
func acquireCacheLock(cacheDir string) (release func(), err error) {
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		return nil, fmt.Errorf("mkdir cache dir: %w", err)
	}
	path := filepath.Join(cacheDir, "refresh.lock")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil, fmt.Errorf("another refresh may already be in progress (remove %s if stale)", path)
		}
		return nil, fmt.Errorf("open lock file: %w", err)
	}
	_ = f.Close()
	return func() { _ = os.Remove(path) }, nil
}
