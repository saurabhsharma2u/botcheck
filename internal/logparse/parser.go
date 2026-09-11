package logparse

import (
	"fmt"
	"net/netip"
	"time"
)

type Entry struct {
	IP        netip.Addr
	Timestamp time.Time
	Method    string
	Path      string
	Status    int
	UA        string
	Referer   string
	Bytes     int64
	Raw       string
}

type LogParser interface {
	Parse(line string) (Entry, error)
	Name() string
}

func GetParser(format string) (LogParser, error) {
	switch format {
	case "nginx", "apache", "auto": // treating all common formats closely for v1
		return NewCommonParser(), nil
	case "json":
		return NewJSONParser(), nil
	case "cloudflare":
		return NewCloudflareParser(), nil
	default:
		return nil, fmt.Errorf("unknown format: %s", format)
	}
}
