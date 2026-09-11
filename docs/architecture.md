# botcheck — Detailed Architecture & Implementation Plan

**Project:** `botcheck`
**Language:** Go
**Version target:** v1 (Registry + Log Scanning)
**Last updated:** 2026-09-11

---

## 1. Project Goals

### v1 Goals
- Maintain a local, frequently-updatable **registry** of known bot / crawler IP ranges (primarily official/good bots).
- Parse common web server access logs.
- Check whether log entries contain IPs that match the registry.
- Produce clear, actionable reports (text + JSON/CSV).
- Provide a solid foundation so future features do **not** require rewriting core packages.

### Non-Goals for v1
- Real-time log tailing
- Residential proxy rotation detection
- User-Agent verification / spoofing detection
- Automatic firewall rule generation
- Web UI or daemon mode
- Paid threat-intelligence APIs (optional later)

---

## 2. Guiding Principles (Avoid Future Rewrites)

| Principle | Why it matters |
|-----------|----------------|
| **Interfaces first** | Every major concern is behind an interface so implementations can be swapped. |
| **Immutable data where possible** | Registry is loaded once and treated as read-only during a scan. |
| **Config-driven sources** | Adding a new IP list never requires code changes. |
| **Efficient matching** | Use `net/netip` + a high-quality prefix tree (longest-prefix match). |
| **Local-first** | All data lives under a user-controlled cache directory. No mandatory remote calls at runtime. |
| **CLI as thin layer** | Business logic lives in `internal/`. Commands only wire dependencies. |
| **Open-source standards from day 1** | Tests, linting, CI, docs, license, semantic versioning. |

---

## 3. High-Level Architecture

```
┌─────────────────┐
│   CLI (Cobra)   │  update | scan | status | check | version
└────────┬────────┘
         │
┌────────▼────────┐
│   Config        │  YAML + flags + env
└────────┬────────┘
         │
┌────────▼──────────────────────────────────────────────┐
│                   Core Domain                         │
│  ┌─────────────┐  ┌─────────────┐  ┌──────────────┐  │
│  │  Source     │  │  Registry   │  │  Matcher     │  │
│  │  (fetcher)  │→ │  (in-memory │→ │  (IP → hit)  │  │
│  └─────────────┘  │   + disk)   │  └──────────────┘  │
│                   └─────────────┘                     │
│  ┌─────────────┐  ┌─────────────┐                     │
│  │  LogParser  │  │  Reporter   │                     │
│  └─────────────┘  └─────────────┘                     │
└───────────────────────────────────────────────────────┘
         │
┌────────▼────────┐
│  Local Storage  │  $XDG_CACHE_HOME/botcheck/ or ~/.cache/botcheck/
│  (manifest +    │  versioned data + metadata
│   prefix sets)  │
└─────────────────┘
```

### Key Interfaces

```go
// Source downloads and normalizes one upstream list
type Source interface {
    Name() string
    Category() string
    Fetch(ctx context.Context) ([]netip.Prefix, Meta, error)
}

// Registry holds the combined, ready-to-query set of prefixes
type Registry interface {
    Load(ctx context.Context) error
    Contains(ip netip.Addr) (Hit, bool)
    Stats() Stats
    LastUpdated() time.Time
}

// LogParser turns a log line into structured records
type LogParser interface {
    Parse(line string) (Entry, error)
    Name() string
}

// Matcher is the pure lookup engine
type Matcher interface {
    Insert(prefix netip.Prefix, meta Meta)
    Lookup(ip netip.Addr) (Meta, bool)
    Len() int
}
```

This separation allows:
- New log formats → new `LogParser`
- New matching algorithm → new `Matcher`
- New upstream lists → config only (or thin `Source` adapter)
- Future rotation detector can consume the same `Entry` stream + `Registry`

---

## 4. Directory Layout

```
botcheck/
├── cmd/
│   └── botcheck/
│       └── main.go                 # thin entrypoint only
├── internal/
│   ├── cli/                        # Cobra commands + flag wiring
│   │   ├── root.go
│   │   ├── update.go
│   │   ├── scan.go
│   │   ├── status.go
│   │   ├── check.go
│   │   └── version.go
│   ├── config/
│   │   └── config.go
│   ├── registry/
│   │   ├── registry.go
│   │   ├── disk.go
│   │   └── manifest.go
│   ├── source/
│   │   ├── source.go               # interface + registry of sources
│   │   ├── http.go
│   │   ├── github.go
│   │   └── static.go
│   ├── matcher/
│   │   └── matcher.go              # wrapper around chosen trie library
│   ├── logparse/
│   │   ├── parser.go
│   │   ├── nginx.go
│   │   ├── apache.go
│   │   ├── json.go
│   │   └── cloudflare.go
│   ├── report/
│   │   ├── text.go
│   │   ├── json.go
│   │   └── csv.go
│   └── version/
│       └── version.go
├── testdata/
│   ├── logs/
│   └── expected/
├── docs/
│   └── architecture.md
├── scripts/
├── .github/
│   └── workflows/
│       ├── ci.yml
│       └── release.yml
├── .golangci.yml
├── go.mod
├── go.sum
├── LICENSE                         # Apache-2.0
├── README.md
├── CONTRIBUTING.md
├── CODE_OF_CONDUCT.md
├── CHANGELOG.md
└── Makefile
```

This follows current Go community consensus (`cmd/` + `internal/` + thin `main`) and scales cleanly.

---

## 5. Core Data Model (v1)

```go
type Meta struct {
    Source    string            // "googlebot", "openai-gptbot", ...
    Category  string            // "search", "ai", "monitoring", "good"
    Name      string            // human-readable name
    UpdatedAt time.Time
    Extra     map[string]string // free-form metadata
}

type Hit struct {
    Prefix netip.Prefix
    Meta   Meta
}

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
    // Future fields can be added without breaking existing parsers
}

type Stats struct {
    TotalPrefixes int
    Sources       map[string]int
    Categories    map[string]int
    LastUpdated   time.Time
}
```

### On-Disk Format (v1)

Location: `$XDG_CACHE_HOME/botcheck/` (fallback `~/.cache/botcheck/`)

```
manifest.json          # version, sources, timestamps, content hash
data/
  prefixes.json.gz     # array of {prefix, source, category, name}
```

Start simple (JSON + gzip). Can switch to a more compact binary format later without changing the public API.

---

## 6. CLI Surface (v1)

```bash
botcheck refresh [--force] [--source NAME] [--config path]
botcheck status
botcheck check <IP> [--json]
botcheck scan <logfile...> [flags]
  --input       auto|nginx|apache|json|cloudflare
  --output      text|json|csv
  --only-matched
  --only-unmatched
  --quiet
  --config      path
botcheck version
```

**Exit codes**
- `0` — success
- `1` — runtime error
- `2` — usage / invalid arguments

---

## 7. Matching Engine

**Recommendation for v1:**

- Use `net/netip` exclusively.
- Prefer a modern, well-maintained library that supports longest-prefix match for both IPv4 and IPv6:

| Library | Notes |
|---------|-------|
| `github.com/gaissmai/bart` | Very fast, actively maintained, excellent choice |
| `github.com/aromatt/netipds` | Immutable collections, clean API |
| Custom thin wrapper | Acceptable for v1 if kept behind the `Matcher` interface |

**Decision rule:** Start with one of the two libraries above. Keep the `Matcher` interface so the implementation can be swapped later with zero impact on the rest of the codebase.

---

## 8. Configuration

**Default config locations (in order):**
1. `--config` flag
2. `./botcheck.yaml`
3. `$XDG_CONFIG_HOME/botcheck/config.yaml`
4. Built-in defaults

**Example `botcheck.yaml`:**

```yaml
cache_dir: ""          # empty = use XDG_CACHE_HOME

# Split setups across files (paths/globs, relative to this file).
# Duplicate `name`s: this file wins over imports.
imports:
  - sources.d/*.yaml

sources:
  - name: my-custom-list
    category: monitoring
    type: http
    url: https://example.com/bot-ips.json
    enabled: true
```

The curated registry lives in `registry/manifest.yaml` (fetched at runtime
from `registry_url`, default `main`). Add a new upstream list there — no
binary release needed. User overrides stay in `botcheck.yaml` / `imports:`;
local entries win over registry ones on duplicate `name`. Offline with a
populated cache, `update` keeps last-good data and `status` shows its age.

Adding a new source should require **only** a config change in the vast majority of cases.

---

## 9. Implementation Phases

### Phase 0 — Project Skeleton (Day 1)
- [ ] `go mod init`
- [ ] Directory structure
- [ ] Cobra root command + `version`
- [ ] Basic `Makefile` (`build`, `test`, `lint`)
- [ ] `.golangci.yml`
- [ ] GitHub Actions CI (test + lint)
- [ ] LICENSE (Apache-2.0), README skeleton, CONTRIBUTING.md

### Phase 1 — Core Domain (Days 2–4)
- [ ] `config` package
- [ ] `matcher` package (wrap chosen trie)
- [ ] `source` interfaces + HTTP source
- [ ] `registry` (in-memory + disk persistence)
- [ ] Unit tests for matcher + registry

### Phase 2 — CLI Commands (Days 5–6)
- [ ] `update` command
- [ ] `status` command
- [ ] `check <IP>` command
- [ ] Integration tests with real (or recorded) sources

### Phase 3 — Log Scanning (Days 7–9)
- [ ] `logparse` package (Nginx + Apache Combined first)
- [ ] `scan` command
- [ ] Text + JSON reporters
- [ ] End-to-end tests with sample logs in `testdata/`

### Phase 4 — Polish & Open Source Readiness (Days 10–12)
- [ ] CSV reporter
- [ ] Better error messages + `--quiet`
- [ ] Documentation (`docs/architecture.md`, expanded README)
- [ ] `goreleaser` configuration
- [ ] Example configs and sample output in README
- [ ] First tagged release (`v0.1.0`)

---

## 10. Open Source Standards Checklist

| Item | Status / Notes |
|------|----------------|
| License | Apache-2.0 (recommended) |
| `go.mod` / `go.sum` | Required |
| Semantic Versioning | Follow from first tag |
| CI | GitHub Actions: test, lint, `govulncheck` |
| Linter | `golangci-lint` with sensible defaults |
| Tests | Unit + integration, coverage target ≥ 70% for core packages |
| Documentation | README + architecture doc + godoc comments |
| CONTRIBUTING.md | How to build, test, add a source, coding style |
| CODE_OF_CONDUCT.md | Contributor Covenant |
| CHANGELOG.md | Keep updated |
| Reproducible builds | `goreleaser` + `ldflags` for version |
| Security | No secrets in repo, minimal dependencies |

---

## 11. Future Extension Points (Already Designed For)

| Feature | How it fits |
|---------|-------------|
| Malicious / abuse IP lists | Additional sources with `category: bad` |
| Residential proxy rotation detection | New analyzer that consumes `[]Entry` + session grouping |
| UA verification (Googlebot spoofing etc.) | New verifier that uses Registry + reverse DNS |
| Real-time tailing | New `watch` command that reuses `LogParser` + `Registry` |
| Firewall export | New reporter that outputs nftables / iptables / Cloudflare rules |
| SQLite backend | Alternative storage behind the same `Registry` interface |

---

## 12. Recommended Dependencies (v1)

**Keep the dependency set small:**

- `github.com/spf13/cobra` — CLI
- `github.com/spf13/viper` *(optional)* or pure YAML
- Chosen prefix-tree library (`bart` or `netipds`)
- `gopkg.in/yaml.v3` (if not using Viper)
- Standard library for everything else (`net/netip`, `net/http`, `compress/gzip`, etc.)

Avoid heavy frameworks.

---

## 13. Testing Strategy

- **Unit tests** next to each package (`*_test.go`)
- **Table-driven tests** for parsers and matchers
- **testdata/** for real log samples and expected output
- **Integration tests** that exercise `update` → `scan` against recorded HTTP responses (use `httptest`)
- **Race detector** in CI (`go test -race`)
- **Coverage** reported in CI

---

## 14. Decision Log (Important Choices)

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Module layout | `cmd/` + `internal/` | Standard, prevents accidental public API |
| IP library | `net/netip` | Modern, immutable, allocation-friendly |
| Prefix matching | Library behind interface | Can swap later without rewrite |
| Config format | YAML | Human-friendly, widely understood |
| Cache location | XDG Base Directory | Follows platform conventions |
| First log formats | Nginx + Apache Combined | Cover the majority of real-world logs |
| License | Apache-2.0 | Clear, patent grant, widely accepted |

---

## 15. Next Immediate Steps

1. Confirm project name (`botcheck` or preferred alternative).
2. Confirm license (Apache-2.0 recommended).
3. Decide on prefix-tree library (`bart` vs `netipds`).
4. Create the repository and implement **Phase 0**.
5. Proceed phase by phase, keeping the interfaces stable.

---

*This document is the single source of truth for the v1 architecture. Any significant deviation should be recorded in the Decision Log above.*
