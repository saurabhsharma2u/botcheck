package logparse

import (
	"encoding/json"
	"fmt"
	"net/netip"
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

	if ipStr, ok := raw["ip"].(string); ok {
		if ip, err := netip.ParseAddr(ipStr); err == nil {
			e.IP = ip
		} else {
			return e, fmt.Errorf("invalid IP: %v", ipStr)
		}
	} else if ipStr, ok := raw["remote_ip"].(string); ok {
		if ip, err := netip.ParseAddr(ipStr); err == nil {
			e.IP = ip
		} else {
			return e, fmt.Errorf("invalid IP: %v", ipStr)
		}
	} else {
		return e, fmt.Errorf("IP field not found")
	}

	if status, ok := raw["status"].(float64); ok {
		e.Status = int(status)
	}

	return e, nil
}
