package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// VerifyMap maps source names to their DNS verification suffixes. It is
// written on every refresh so --dns can fall back to reverse-DNS
// verification for IPs with no range match, without fetching the manifest
// (check/scan stay offline-first).
type VerifyMap struct {
	Version int                 `json:"version"`
	Sources map[string][]string `json:"sources"`
}

func verifyPath(cacheDir string) string {
	return filepath.Join(cacheDir, "verify.json")
}

// SaveVerifyMap persists the per-source verification suffixes next to the
// cached registry data. A nil map writes an empty (but present) file so
// readers can distinguish "no verifiable sources" from "stale cache".
func SaveVerifyMap(cacheDir string, sources map[string][]string) error {
	if sources == nil {
		sources = make(map[string][]string)
	}
	b, err := json.Marshal(VerifyMap{Version: 1, Sources: sources})
	if err != nil {
		return err
	}
	if err := os.WriteFile(verifyPath(cacheDir), b, 0o600); err != nil {
		return fmt.Errorf("save verify map: %w", err)
	}
	return nil
}

// LoadVerifyMap reads the persisted suffix map. os.ErrNotExist means the
// cache predates DNS verification support (refresh once to create it).
func LoadVerifyMap(cacheDir string) (map[string][]string, error) {
	b, err := os.ReadFile(verifyPath(cacheDir))
	if err != nil {
		return nil, err
	}
	var m VerifyMap
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("parse verify map: %w", err)
	}
	if m.Sources == nil {
		m.Sources = make(map[string][]string)
	}
	return m.Sources, nil
}
