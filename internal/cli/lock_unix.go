//go:build darwin || linux

package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// acquireCacheLock takes an exclusive, non-blocking flock on
// <cacheDir>/refresh.lock so two scheduled refreshes (systemd timer,
// launchd, cron) can never rewrite the cache concurrently. The lock is
// released when the returned function is called; the kernel also drops it
// if the process dies, so there is no stale-lock state to clean up.
func acquireCacheLock(cacheDir string) (release func(), err error) {
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		return nil, fmt.Errorf("mkdir cache dir: %w", err)
	}
	path := filepath.Join(cacheDir, "refresh.lock")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open lock file: %w", err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("another refresh is already in progress: %w", err)
	}
	return func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
	}, nil
}
