//go:build linux

package scheduler

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	serviceName = "botcheck-refresh.service"
	timerName   = "botcheck-refresh.timer"
)

// unitDir returns the systemd user unit directory.
func unitDir() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "systemd", "user"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home dir: %w", err)
	}
	return filepath.Join(home, ".config", "systemd", "user"), nil
}

// writeUnits renders and writes the service + timer files, returning paths.
func writeUnits(dir string, o Options) ([]string, error) {
	data, err := o.data()
	if err != nil {
		return nil, err
	}
	svc, err := render("systemd-service.tmpl", data)
	if err != nil {
		return nil, err
	}
	tmr, err := render("systemd-timer.tmpl", data)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir %s: %w", dir, err)
	}
	paths := []string{filepath.Join(dir, serviceName), filepath.Join(dir, timerName)}
	contents := []string{svc, tmr}
	for i, p := range paths {
		if err := os.WriteFile(p, []byte(contents[i]), 0o644); err != nil {
			return nil, fmt.Errorf("write %s: %w", p, err)
		}
	}
	return paths, nil
}

func enable() error {
	cmds := [][]string{
		{"systemctl", "--user", "daemon-reload"},
		{"systemctl", "--user", "enable", "--now", timerName},
	}
	for _, c := range cmds {
		if out, err := exec.Command(c[0], c[1:]...).CombinedOutput(); err != nil {
			return fmt.Errorf("%s: %w\n%s", strings.Join(c, " "), err, strings.TrimSpace(string(out)))
		}
	}
	return nil
}

func disable() error {
	out, err := exec.Command("systemctl", "--user", "disable", "--now", timerName).CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemctl --user disable --now %s: %w\n%s", timerName, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Install writes user-level systemd units and enables the timer.
func Install(o Options) (string, error) {
	dir, err := unitDir()
	if err != nil {
		return "", err
	}
	paths, err := writeUnits(dir, o)
	if err != nil {
		return "", err
	}
	done := fmt.Sprintf("Wrote:\n  %s\n", strings.Join(paths, "\n  "))
	if err := enable(); err != nil {
		return done, fmt.Errorf("auto-enable failed: %w", err)
	}
	return done + "Enabled with: systemctl --user enable --now " + timerName, nil
}

// Uninstall disables the timer and removes the unit files.
func Uninstall() (string, error) {
	dir, err := unitDir()
	if err != nil {
		return "", err
	}
	// Best effort: unit files may already be gone.
	_ = disable()
	var removed []string
	for _, n := range []string{serviceName, timerName} {
		p := filepath.Join(dir, n)
		if err := os.Remove(p); err == nil {
			removed = append(removed, p)
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("remove %s: %w", p, err)
		}
	}
	if len(removed) == 0 {
		return "Nothing installed.", nil
	}
	return "Removed:\n  " + strings.Join(removed, "\n  "), nil
}

// Status reports installed units plus timer state (best effort).
func Status() (string, error) {
	dir, err := unitDir()
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	for _, n := range []string{serviceName, timerName} {
		p := filepath.Join(dir, n)
		if _, err := os.Stat(p); err == nil {
			fmt.Fprintf(&sb, "installed: %s\n", p)
		} else {
			fmt.Fprintf(&sb, "missing:   %s\n", p)
		}
	}
	if out, err := exec.Command("systemctl", "--user", "is-enabled", timerName).CombinedOutput(); err == nil {
		fmt.Fprintf(&sb, "enabled: %s\n", strings.TrimSpace(string(out)))
	}
	if out, err := exec.Command("systemctl", "--user", "is-active", timerName).CombinedOutput(); err == nil {
		fmt.Fprintf(&sb, "active:  %s\n", strings.TrimSpace(string(out)))
	}
	return strings.TrimRight(sb.String(), "\n"), nil
}
