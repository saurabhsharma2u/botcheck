# botcheck

A high-performance IP prefix matcher and log scanner for known bot / crawler IPs.

## Features

- Fast IP matching using longest prefix match (`netip` and `bart` trie).
- Maintains a local registry of known IP lists (e.g. Googlebot, Bingbot).
- Scans common log formats (Nginx/Apache, JSON) to check for bots.
- Flexible outputs (Text, JSON, CSV).

## Installation

```bash
go install github.com/saurabhsharma2u/iambot/cmd/botcheck@latest
```

## Configuration

No config needed to start: `botcheck update` fetches the curated registry
(`registry/manifest.yaml` on `main`: Googlebot, Bingbot, Applebot, OpenAI,
Anthropic, Perplexity, DuckDuckGo, Ahrefs, Meta and more).
`botcheck status` shows what's loaded, including the data version.
Offline with a populated cache, `update` keeps the last-good data.

To customize, create a `botcheck.yaml` file:

```yaml
cache_dir: "/tmp/botcheck-cache"

# Pin the registry (branch, tag, or SHA; default main) or point at a mirror.
registry_ref: "main"
# registry_url: "off"  # disable registry, local sources only

# Split large setups across files (paths/globs, relative to this file).
imports:
  - sources.d/*.yaml

sources:
  # Local entries override registry ones on duplicate `name`.
  - name: my-custom-list
    category: monitoring
    type: http
    url: https://example.com/bot-ips.json
    enabled: true

  # Or a maintained local file (one CIDR/IP per line, # comments allowed).
  - name: facebook
    category: social
    type: file
    path: /path/to/meta-prefixes.txt
    enabled: true
```

## Usage

Update the registry:
```bash
botcheck update
```

Check the registry status:
```bash
botcheck status
```

Check a single IP:
```bash
botcheck check 192.168.1.1
```

Scan log files:
```bash
botcheck scan access.log --format auto --output text
botcheck scan access.log --only-matched --output json
```

## License
Apache-2.0
