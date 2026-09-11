package logparse

import (
	"fmt"
	"net/netip"
	"strconv"
	"strings"
)

// A simplistic common log format parser for Apache/Nginx combined logs.
// Format typically: IP - user [time] "method path proto" status bytes "referer" "UA"

type commonParser struct{}

func NewCommonParser() LogParser {
	return &commonParser{}
}

func (p *commonParser) Name() string {
	return "common"
}

func (p *commonParser) Parse(line string) (Entry, error) {
	e := Entry{Raw: line}

	// Extremely naive parsing for speed and simplicity in v1.
	// Real implementation should use a proper regex or state machine.
	parts := strings.SplitN(line, " ", 2)
	if len(parts) < 2 {
		return e, fmt.Errorf("invalid format")
	}

	ip, err := netip.ParseAddr(parts[0])
	if err != nil {
		return e, fmt.Errorf("parse IP: %w", err)
	}
	e.IP = ip

	// For v1, we won't fully parse all fields of Apache log, just need IP.
	// We'll try to extract UA and Status if possible loosely.

	if statusIdx := strings.LastIndex(line, "\" "); statusIdx != -1 {
		statusParts := strings.Split(line[statusIdx+2:], " ")
		if len(statusParts) >= 2 {
			if s, err := strconv.Atoi(statusParts[0]); err == nil {
				e.Status = s
			}
			if b, err := strconv.ParseInt(statusParts[1], 10, 64); err == nil {
				e.Bytes = b
			}
		}
	}

	return e, nil
}
