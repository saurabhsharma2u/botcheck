//go:build !linux && !darwin

package scheduler

import "errors"

// Scheduled refresh is supported on Linux (systemd) and macOS (launchd) only.
func Install(o Options) (string, error) {
	return "", errors.New("scheduled refresh is supported on linux and macOS only")
}

func Uninstall() (string, error) {
	return "", errors.New("scheduled refresh is supported on linux and macOS only")
}

func Status() (string, error) {
	return "", errors.New("scheduled refresh is supported on linux and macOS only")
}
