package scheduler

import (
	"io/fs"
	"strings"
	"testing"
)

func TestParseAt(t *testing.T) {
	for _, tc := range []struct {
		in      string
		hour    string
		minute  string
		wantErr bool
	}{
		{"03:17", "03", "17", false},
		{"3:7", "03", "07", false},
		{"23:59", "23", "59", false},
		{"00:00", "00", "00", false},
		{"24:00", "", "", true},
		{"03:60", "", "", true},
		{"0317", "", "", true},
		{"", "", "", true},
		{"ab:cd", "", "", true},
	} {
		h, m, err := parseAt(tc.in)
		if tc.wantErr && err == nil {
			t.Errorf("parseAt(%q): expected error, got nil", tc.in)
		}
		if !tc.wantErr {
			if err != nil {
				t.Errorf("parseAt(%q): unexpected error: %v", tc.in, err)
			} else if h != tc.hour || m != tc.minute {
				t.Errorf("parseAt(%q) = %s:%s, want %s:%s", tc.in, h, m, tc.hour, tc.minute)
			}
		}
	}
}

func TestRenderAllTemplates(t *testing.T) {
	data := unitData{Binary: "/usr/local/bin/botcheck", Config: "/etc/botcheck/botcheck.yaml", Hour: "03", Minute: "17"}
	entries, err := templatesFS.ReadDir("templates")
	if err != nil {
		t.Fatalf("read templates dir: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("no embedded templates found")
	}
	for _, e := range entries {
		out, err := render(e.Name(), data)
		if err != nil {
			t.Errorf("render %s: %v", e.Name(), err)
			continue
		}
		if e.Name() != "systemd-timer.tmpl" && !strings.Contains(out, data.Binary) {
			t.Errorf("render %s: missing binary path", e.Name())
		}
		if e.Name() != "systemd-service.tmpl" &&
			(!strings.Contains(out, "03") || !strings.Contains(out, "17")) {
			t.Errorf("render %s: missing schedule time", e.Name())
		}
		if strings.Contains(out, "{{") {
			t.Errorf("render %s: unrendered placeholder", e.Name())
		}
		var _ fs.DirEntry = e
	}
}

func TestRenderOmitsEmptyConfig(t *testing.T) {
	data := unitData{Binary: "/usr/local/bin/botcheck", Hour: "03", Minute: "17"}
	for _, name := range []string{"systemd-service.tmpl", "launchd-plist.tmpl"} {
		out, err := render(name, data)
		if err != nil {
			t.Fatalf("render %s: %v", name, err)
		}
		if strings.Contains(out, "--config") {
			t.Errorf("render %s without config: unexpected --config flag", name)
		}
	}
}
