package logparse

import (
	"encoding/json"
	"fmt"
	"net/netip"
	"strconv"
	"time"
)

type jsonParser struct{}

func NewJSONParser() LogParser {
	return &jsonParser{}
}

func (p *jsonParser) Name() string {
	return "json"
}

func (p *jsonParser) Parse(line string) (Entry, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return Entry{Raw: line}, err
	}

	e := Entry{Raw: line}

	ipStr, ok := firstString(raw, "ip", "remote_ip", "remoteIp", "client_ip", "clientIp", "src_ip", "srcIp")
	if !ok {
		return e, fmt.Errorf("IP field not found")
	}
	ip, err := netip.ParseAddr(ipStr)
	if err != nil {
		return e, fmt.Errorf("invalid IP: %v", ipStr)
	}
	e.IP = ip

	if status, ok := raw["status"]; ok {
		switch v := status.(type) {
		case float64:
			e.Status = int(v)
		case string:
			if s, err := strconv.Atoi(v); err == nil {
				e.Status = s
			}
		}
	}

	e.Method, _ = firstString(raw, "method")
	e.Path, _ = firstString(raw, "path", "uri", "url", "request_uri")
	e.UA, _ = firstString(raw, "ua", "user_agent", "userAgent")
	e.Referer, _ = firstString(raw, "referer", "referrer")

	if ts, ok := firstString(raw, "timestamp", "time", "datetime"); ok {
		e.Timestamp = parseTimestamp(ts)
	} else if epoch, ok := raw["timestamp"]; ok {
		if f, ok := epoch.(float64); ok {
			e.Timestamp = unixAuto(int64(f))
		}
	}

	return e, nil
}

func firstString(raw map[string]interface{}, keys ...string) (string, bool) {
	for _, k := range keys {
		if v, ok := raw[k].(string); ok && v != "" {
			return v, true
		}
	}
	return "", false
}

func parseTimestamp(s string) time.Time {
	for _, layout := range []string{
		time.RFC3339,
		"2006-01-02T15:04:05.999999999Z07:00",
		"02/Jan/2006:15:04:05 -0700",
		"2006-01-02 15:04:05",
	} {
		if ts, err := time.Parse(layout, s); err == nil {
			return ts
		}
	}
	if epoch, err := strconv.ParseInt(s, 10, 64); err == nil {
		return unixAuto(epoch)
	}
	return time.Time{}
}

func unixAuto(v int64) time.Time {
	if v > 1e11 {
		return time.Unix(v/1000, (v%1000)*1e6)
	}
	return time.Unix(v, 0)
}
