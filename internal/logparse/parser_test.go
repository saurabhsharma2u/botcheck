package logparse

import (
	"testing"
)

func TestCommonParser(t *testing.T) {
	p := NewCommonParser()
	line := `127.0.0.1 - - [10/Oct/2000:13:55:36 -0700] "GET /apache_pb.gif HTTP/1.0" 200 2326 "http://www.example.com/" "Mozilla/4.08 [en] (Win98; I ;Nav)"`

	e, err := p.Parse(line)
	if err != nil {
		t.Fatal(err)
	}
	if e.IP.String() != "127.0.0.1" {
		t.Errorf("expected 127.0.0.1, got %s", e.IP.String())
	}
}

func TestCommonParserFields(t *testing.T) {
	p := NewCommonParser()
	line := `205.234.147.102 - - [11/Sep/2026:00:08:19 +0000] "GET /country/india HTTP/2.0" 200 30991 "https://ref.example/" "TestAgent/1.0"`

	e, err := p.Parse(line)
	if err != nil {
		t.Fatal(err)
	}
	if e.Method != "GET" {
		t.Errorf("expected GET, got %q", e.Method)
	}
	if e.Path != "/country/india" {
		t.Errorf("expected /country/india, got %q", e.Path)
	}
	if e.Status != 200 {
		t.Errorf("expected 200, got %d", e.Status)
	}
	if e.Bytes != 30991 {
		t.Errorf("expected 30991, got %d", e.Bytes)
	}
	if e.Referer != "https://ref.example/" {
		t.Errorf("expected referer, got %q", e.Referer)
	}
	if e.UA != "TestAgent/1.0" {
		t.Errorf("expected UA, got %q", e.UA)
	}
	if e.Timestamp.Year() != 2026 || e.Timestamp.Month() != 9 || e.Timestamp.Day() != 11 {
		t.Errorf("bad timestamp: %v", e.Timestamp)
	}
}

func TestCommonParserMinimal(t *testing.T) {
	p := NewCommonParser()
	e, err := p.Parse("8.8.8.8 - - [11/Sep/2026:00:08:19 +0000] \"-\" 400 -")
	if err != nil {
		t.Fatal(err)
	}
	if e.Method != "" || e.Path != "" {
		t.Errorf("expected empty method/path, got %q %q", e.Method, e.Path)
	}
	if e.Status != 400 {
		t.Errorf("expected 400, got %d", e.Status)
	}
	if e.UA != "" || e.Referer != "" {
		t.Errorf("expected empty UA/referer, got %q %q", e.UA, e.Referer)
	}
}

func TestCommonParserBracketedIPv6(t *testing.T) {
	p := NewCommonParser()
	e, err := p.Parse(`[2001:db8::1] - - [11/Sep/2026:00:08:19 +0000] "GET / HTTP/1.1" 200 10 "-" "-"`)
	if err != nil {
		t.Fatal(err)
	}
	if e.IP.String() != "2001:db8::1" {
		t.Errorf("expected 2001:db8::1, got %s", e.IP.String())
	}
}

func TestCommonParserBadIP(t *testing.T) {
	p := NewCommonParser()
	if _, err := p.Parse("not-an-ip - -"); err == nil {
		t.Error("expected error for bad IP, got nil")
	}
}

func TestJSONParser(t *testing.T) {
	p := NewJSONParser()
	line := `{"ip": "192.168.1.1", "status": 200}`

	e, err := p.Parse(line)
	if err != nil {
		t.Fatal(err)
	}
	if e.IP.String() != "192.168.1.1" {
		t.Errorf("expected 192.168.1.1, got %s", e.IP.String())
	}
	if e.Status != 200 {
		t.Errorf("expected 200, got %d", e.Status)
	}
}

func TestJSONParserVariants(t *testing.T) {
	p := NewJSONParser()
	e, err := p.Parse(`{"remote_ip": "10.0.0.1", "status": "404", "method": "POST", "path": "/x", "user_agent": "UA", "referer": "R", "timestamp": "2026-09-11T00:00:00Z"}`)
	if err != nil {
		t.Fatal(err)
	}
	if e.IP.String() != "10.0.0.1" || e.Status != 404 {
		t.Errorf("bad ip/status: %v %d", e.IP, e.Status)
	}
	if e.Method != "POST" || e.Path != "/x" || e.UA != "UA" || e.Referer != "R" {
		t.Errorf("bad fields: %+v", e)
	}
	if e.Timestamp.IsZero() {
		t.Error("expected timestamp parsed")
	}
}

func TestJSONParserEpochMillis(t *testing.T) {
	p := NewJSONParser()
	e, err := p.Parse(`{"ip": "10.0.0.2", "timestamp": 1757548800000}`)
	if err != nil {
		t.Fatal(err)
	}
	if e.Timestamp.Year() != 2025 || e.Timestamp.Month() != 9 || e.Timestamp.Day() != 11 {
		t.Errorf("bad millis timestamp: %v", e.Timestamp)
	}

	e, err = p.Parse(`{"ip": "10.0.0.2", "timestamp": "1757548800"}`)
	if err != nil {
		t.Fatal(err)
	}
	if e.Timestamp.Year() != 2025 {
		t.Errorf("bad epoch timestamp: %v", e.Timestamp)
	}
}

func TestCloudflareParser(t *testing.T) {
	p := NewCloudflareParser()
	line := `{"ClientIP": "1.2.3.4", "ClientRequestMethod": "GET", "ClientRequestURI": "/", "EdgeResponseStatus": 200, "ClientRequestUserAgent": "bot", "EdgeStartTimestamp": "2026-09-11T00:00:00Z"}`
	e, err := p.Parse(line)
	if err != nil {
		t.Fatal(err)
	}
	if e.IP.String() != "1.2.3.4" || e.Method != "GET" || e.UA != "bot" {
		t.Errorf("bad entry: %+v", e)
	}

	if _, err := p.Parse(`{"ClientIP": "nope"}`); err == nil {
		t.Error("expected error for invalid ClientIP, got nil")
	}

	e, err = p.Parse(`{"ip": "5.6.7.8"}`)
	if err != nil {
		t.Fatal(err)
	}
	if e.IP.String() != "5.6.7.8" {
		t.Errorf("expected generic fallback 5.6.7.8, got %s", e.IP.String())
	}
}
