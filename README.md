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

Create a `botcheck.yaml` file:

```yaml
cache_dir: "/tmp/botcheck-cache"

sources:
  - name: googlebot
    category: search
    type: http
    url: https://developers.google.com/static/search/apis/ipranges/googlebot.json
    enabled: true
  - name: google-common-crawlers
    category: search
    type: http
    url: https://developers.google.com/static/crawling/ipranges/common-crawlers.json
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
