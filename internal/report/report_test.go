package report

import (
	"bytes"
	"encoding/json"
	"net/netip"
	"strings"
	"testing"

	"github.com/saurabhsharma2u/iambot/internal/logparse"
	"github.com/saurabhsharma2u/iambot/internal/matcher"
)

func testHit() Result {
	return Result{
		Entry: logparse.Entry{
			IP:      netip.MustParseAddr("1.1.1.1"),
			Method:  "GET",
			Path:    "/",
			Status:  200,
			Bytes:   100,
			UA:      "bot",
			Referer: "https://ref.example/",
		},
		Match: true,
		Hit: matcher.Hit{
			Prefix: netip.MustParsePrefix("1.1.1.0/24"),
			Meta:   matcher.Meta{Source: "cloudflare", Category: "infra", Name: "cf"},
		},
	}
}

func TestTextReporter(t *testing.T) {
	r := &TextReporter{}

	var buf bytes.Buffer
	if err := r.Report(&buf, testHit()); err != nil {
		t.Fatal(err)
	}

	expected := "[MATCH] 1.1.1.1\n" +
		"  prefix:   1.1.1.0/24 (cloudflare/infra)\n" +
		"  time: - | request: GET / -> 200 (100 bytes) | referer: https://ref.example/\n" +
		"  ua:       bot\n"
	if buf.String() != expected {
		t.Errorf("expected %q, got %q", expected, buf.String())
	}
	if strings.Contains(buf.String(), "\x1b") {
		t.Errorf("expected no color codes for non-terminal writer, got %q", buf.String())
	}
	if useColor(&buf) {
		t.Error("expected useColor false for bytes.Buffer")
	}
}

func TestTextReporterMiss(t *testing.T) {
	r := &TextReporter{}
	res := Result{
		Entry: logparse.Entry{IP: netip.MustParseAddr("8.8.8.8")},
		Match: false,
	}

	var buf bytes.Buffer
	if err := r.Report(&buf, res); err != nil {
		t.Fatal(err)
	}

	expected := "[MISS] 8.8.8.8\n" +
		"  time: - | request: - | referer: -\n" +
		"  ua:       -\n"
	if buf.String() != expected {
		t.Errorf("expected %q, got %q", expected, buf.String())
	}
}

func TestJSONReporter(t *testing.T) {
	r := &JSONReporter{}

	var buf bytes.Buffer
	if err := r.Report(&buf, testHit()); err != nil {
		t.Fatal(err)
	}

	var out map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	for k, want := range map[string]interface{}{
		"ip": "1.1.1.1", "method": "GET", "path": "/", "status": float64(200),
		"bytes": float64(100), "ua": "bot", "referer": "https://ref.example/",
		"matched": true, "prefix": "1.1.1.0/24", "source": "cloudflare",
		"category": "infra", "name": "cf",
	} {
		if out[k] != want {
			t.Errorf("expected %s=%v, got %v", k, want, out[k])
		}
	}
	if _, ok := out["timestamp"]; ok {
		t.Error("expected no timestamp for zero time")
	}
	if err := r.Flush(); err != nil {
		t.Fatal(err)
	}
}

func TestCSVReporter(t *testing.T) {
	r := &CSVReporter{}

	var buf bytes.Buffer
	if err := r.Report(&buf, testHit()); err != nil {
		t.Fatal(err)
	}
	if err := r.Flush(); err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected header + 1 row, got %q", buf.String())
	}
	if lines[0] != "timestamp,ip,matched,prefix,source,category,method,path,status,bytes,ua,referer" {
		t.Errorf("bad header: %q", lines[0])
	}
	if !strings.Contains(lines[1], "1.1.1.0/24") || !strings.Contains(lines[1], "GET") {
		t.Errorf("bad row: %q", lines[1])
	}
}
