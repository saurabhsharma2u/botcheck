package report

import (
	"bytes"
	"net/netip"
	"testing"

	"github.com/saurabhsharma2u/iambot/internal/logparse"
	"github.com/saurabhsharma2u/iambot/internal/matcher"
)

func TestTextReporter(t *testing.T) {
	r := &TextReporter{}
	res := Result{
		Entry: logparse.Entry{IP: netip.MustParseAddr("1.1.1.1")},
		Match: true,
		Hit: matcher.Hit{
			Meta: matcher.Meta{Source: "cloudflare"},
		},
	}

	var buf bytes.Buffer
	if err := r.Report(&buf, res); err != nil {
		t.Fatal(err)
	}

	expected := "[MATCH] 1.1.1.1 cloudflare\n"
	if buf.String() != expected {
		t.Errorf("expected %q, got %q", expected, buf.String())
	}
}
