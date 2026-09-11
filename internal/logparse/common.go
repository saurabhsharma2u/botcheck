package logparse

import (
	"fmt"
	"net/netip"
	"strconv"
	"strings"
	"time"
)

type commonParser struct{}

func NewCommonParser() LogParser {
	return &commonParser{}
}

func (p *commonParser) Name() string {
	return "common"
}

func (p *commonParser) Parse(line string) (Entry, error) {
	e := Entry{Raw: line}

	ipStr, rest, found := strings.Cut(line, " ")
	if !found {
		return e, fmt.Errorf("invalid format")
	}
	ip, err := netip.ParseAddr(ipStr)
	if err != nil {
		if trimmed, ok := strings.CutPrefix(ipStr, "["); ok {
			if end := strings.IndexByte(trimmed, ']'); end != -1 {
				ip, err = netip.ParseAddr(trimmed[:end])
			}
		}
		if err != nil {
			return e, fmt.Errorf("parse IP: %w", err)
		}
	}
	e.IP = ip

	if i := strings.IndexByte(rest, '['); i != -1 {
		if j := strings.IndexByte(rest[i:], ']'); j != -1 {
			if ts, err := time.Parse("02/Jan/2006:15:04:05 -0700", rest[i+1:i+j]); err == nil {
				e.Timestamp = ts
			}
			rest = rest[i+j+1:]
		}
	}

	if i := strings.IndexByte(rest, '"'); i != -1 {
		rest = rest[i+1:]
		if j := strings.IndexByte(rest, '"'); j != -1 {
			if req := rest[:j]; req != "-" {
				parts := strings.SplitN(req, " ", 3)
				if len(parts) > 0 {
					e.Method = parts[0]
				}
				if len(parts) > 1 {
					e.Path = parts[1]
				}
			}
			rest = rest[j+1:]
		}
	}

	fields := strings.Fields(rest)
	if len(fields) > 0 && fields[0] != "-" {
		if s, err := strconv.Atoi(fields[0]); err == nil {
			e.Status = s
		}
	}
	if len(fields) > 1 && fields[1] != "-" {
		if b, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
			e.Bytes = b
		}
	}

	referer, ua := quotedPair(rest)
	if referer != "-" {
		e.Referer = referer
	}
	if ua != "-" {
		e.UA = ua
	}
	return e, nil
}

func quotedPair(s string) (first, second string) {
	i := strings.IndexByte(s, '"')
	if i == -1 {
		return "", ""
	}
	s = s[i+1:]
	j := strings.IndexByte(s, '"')
	if j == -1 {
		return "", ""
	}
	first = s[:j]
	s = s[j+1:]
	i = strings.IndexByte(s, '"')
	if i == -1 {
		return first, ""
	}
	s = s[i+1:]
	if j := strings.IndexByte(s, '"'); j != -1 {
		return first, s[:j]
	}
	return first, s
}
