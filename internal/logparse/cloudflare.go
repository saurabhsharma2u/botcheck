package logparse

import (
	"encoding/json"
	"fmt"
	"net/netip"
	"strconv"
)

type cloudflareParser struct {
	jsonParser *jsonParser
}

func NewCloudflareParser() LogParser {
	return &cloudflareParser{jsonParser: &jsonParser{}}
}

func (p *cloudflareParser) Name() string {
	return "cloudflare"
}

func (p *cloudflareParser) Parse(line string) (Entry, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return Entry{Raw: line}, err
	}

	if _, ok := raw["ClientIP"]; ok {
		e := parseCloudflare(raw, line)
		if !e.IP.IsValid() {
			return e, fmt.Errorf("invalid ClientIP")
		}
		return e, nil
	}
	return p.jsonParser.Parse(line)
}

func parseCloudflare(raw map[string]interface{}, line string) Entry {
	e := Entry{Raw: line}

	if ipStr, ok := raw["ClientIP"].(string); ok {
		if ip, err := netip.ParseAddr(ipStr); err == nil {
			e.IP = ip
		}
	}
	e.Method, _ = firstString(raw, "ClientRequestMethod")
	e.Path, _ = firstString(raw, "ClientRequestURI")
	e.UA, _ = firstString(raw, "ClientRequestUserAgent")
	e.Referer, _ = firstString(raw, "ClientRequestReferer")

	switch v := raw["EdgeResponseStatus"].(type) {
	case float64:
		e.Status = int(v)
	case string:
		if s, err := strconv.Atoi(v); err == nil {
			e.Status = s
		}
	}

	if ts, ok := raw["EdgeStartTimestamp"].(string); ok {
		e.Timestamp = parseTimestamp(ts)
	} else if f, ok := raw["EdgeStartTimestamp"].(float64); ok {
		e.Timestamp = parseTimestamp(strconv.FormatInt(int64(f), 10))
	}

	return e
}
