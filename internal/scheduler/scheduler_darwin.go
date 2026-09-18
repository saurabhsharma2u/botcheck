//go:build darwin

package scheduler

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const agentLabel = "com.botcheck.refresh"

// unitDir returns the per-user LaunchAgents directory.
func unitDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home dir: %w", err)
	}
	return filepath.Join(home, "Library", "LaunchAgents"), nil
}

func plistPath(dir string) string {
	return filepath.Join(dir, agentLabel+".plist")
}

// writeUnits renders and writes the LaunchAgent plist, returning its path.
func writeUnits(dir string, o Options) ([]string, error) {
	data, err := o.data()
	if err != nil {
		return nil, err
	}
	// launchd wants bare integers, not zero-padded strings.
	data.Hour = strings.TrimLeft(data.Hour, "0")
	data.Minute = strings.TrimLeft(data.Minute, "0")
	if data.Hour == "" {
		data.Hour = "0"
	}
	if data.Minute == "" {
		data.Minute = "0"
	}
	plist, err := render("launchd-plist.tmpl", data)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir %s: %w", dir, err)
	}
	p := plistPath(dir)
	if err := os.WriteFile(p, []byte(plist), 0o644); err != nil {
		return nil, fmt.Errorf("write %s: %w", p, err)
	}
	return []string{p}, nil
}

func domain() string {
	return fmt.Sprintf("gui/%d", os.Getuid())
}

func enable() error {
	dir, err := unitDir()
	if err != nil {
		return err
	}
	// Unload first so reinstalls don't fail with "already loaded".
	_ = exec.Command("launchctl", "bootout", domain(), plistPath(dir)).Run()
	if out, err := exec.Command("launchctl", "bootstrap", domain(), plistPath(dir)).CombinedOutput(); err != nil {
		return fmt.Errorf("launchctl bootstrap: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func disable() error {
	dir, err := unitDir()
	if err != nil {
		return err
	}
	out, err := exec.Command("launchctl", "bootout", domain(), plistPath(dir)).CombinedOutput()
	if err != nil && !strings.Contains(string(out), "No such process") {
		return fmt.Errorf("launchctl bootout: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Install writes the LaunchAgent plist and loads it.
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
		return done, fmt.Errorf("auto-load failed: %w\nLoad manually with: launchctl bootstrap %s %s",
			err, domain(), paths[0])
	}
	return done + "Loaded with: launchctl bootstrap " + domain() + " " + paths[0], nil
}

// Uninstall unloads the agent and removes the plist.
func Uninstall() (string, error) {
	dir, err := unitDir()
	if err != nil {
		return "", err
	}
	// Best effort: the plist may already be gone.
	_ = disable()
	p := plistPath(dir)
	if err := os.Remove(p); err != nil {
		if os.IsNotExist(err) {
			return "Nothing installed.", nil
		}
		return "", fmt.Errorf("remove %s: %w", p, err)
	}
	return "Removed:\n  " + p, nil
}

// Status reports the installed plist plus agent state (best effort).
func Status() (string, error) {
	dir, err := unitDir()
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	p := plistPath(dir)
	if _, err := os.Stat(p); err == nil {
		fmt.Fprintf(&sb, "installed: %s\n", p)
	} else {
		fmt.Fprintf(&sb, "missing:   %s\n", p)
	}
	if out, err := exec.Command("launchctl", "list", agentLabel).CombinedOutput(); err == nil {
		fmt.Fprintf(&sb, "loaded:\n%s", string(out))
	} else {
		fmt.Fprintf(&sb, "loaded: no\n")
	}
	return strings.TrimRight(sb.String(), "\n"), nil
}
