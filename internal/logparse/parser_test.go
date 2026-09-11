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
