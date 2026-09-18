// Package scheduler installs OS-native periodic refresh for botcheck:
// a systemd user timer on Linux, a LaunchAgent on macOS. Unit templates
// are embedded in the binary so every install method (go install,
// release tarball, Homebrew) can set up scheduling without extra files.
package scheduler

import (
	"embed"
	"fmt"
	"strconv"
	"strings"
	"text/template"
)

//go:embed templates/*
var templatesFS embed.FS

// Options controls what gets written into the scheduled unit.
type Options struct {
	// Binary is the absolute path to the botcheck executable.
	Binary string
	// Config is an optional --config path baked into the unit.
	// Empty means run `botcheck refresh` with default config discovery.
	Config string
	// At is the daily run time as "HH:MM".
	At string
}

// unitData is the template data shared by all unit templates.
type unitData struct {
	Binary string
	Config string
	Hour   string
	Minute string
}

func (o Options) data() (unitData, error) {
	if o.Binary == "" {
		return unitData{}, fmt.Errorf("binary path is empty")
	}
	h, m, err := parseAt(o.At)
	if err != nil {
		return unitData{}, err
	}
	return unitData{Binary: o.Binary, Config: o.Config, Hour: h, Minute: m}, nil
}

// parseAt validates "HH:MM" and returns zero-padded hour/minute strings
// (launchd wants bare integers, so callers trim as needed).
func parseAt(at string) (hour, minute string, err error) {
	invalid := func() (string, string, error) {
		return "", "", fmt.Errorf("invalid time %q, want HH:MM (24h)", at)
	}
	parts := strings.Split(at, ":")
	if len(parts) != 2 {
		return invalid()
	}
	h, err1 := strconv.Atoi(parts[0])
	m, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return invalid()
	}
	return fmt.Sprintf("%02d", h), fmt.Sprintf("%02d", m), nil
}

func render(name string, data unitData) (string, error) {
	b, err := templatesFS.ReadFile("templates/" + name)
	if err != nil {
		return "", fmt.Errorf("read template %s: %w", name, err)
	}
	t, err := template.New(name).Parse(string(b))
	if err != nil {
		return "", fmt.Errorf("parse template %s: %w", name, err)
	}
	var sb strings.Builder
	if err := t.Execute(&sb, data); err != nil {
		return "", fmt.Errorf("render template %s: %w", name, err)
	}
	return sb.String(), nil
}
