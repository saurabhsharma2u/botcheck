# botcheck

Find known bots in your logs. `botcheck` matches visitor IPs against a
curated registry of verified crawler ranges — Googlebot, Bingbot, GPTBot,
Applebot, Meta and more — at ~1M log lines per second.

## Quick start

```bash
go install github.com/saurabhsharma2u/iambot/cmd/botcheck@latest

botcheck refresh
botcheck scan /var/log/nginx/access.log --only-matched
```

That's it — no config needed. `update` fetches the curated registry,
`scan` prints every log line that came from a known bot:

```
[MATCH] 40.77.167.61 bingbot
[MATCH] 66.249.79.132 google-common-crawlers
```

## Commands

| Command | What it does |
|---|---|
| `botcheck refresh` | Download the latest bot IP ranges into the local cache |
| `botcheck check 40.77.167.61` | Check a single IP (`--json` for machine output) |
| `botcheck scan access.log` | Check every IP in a log file (`--only-matched`, `--output text\|json\|csv`, `--format nginx\|apache\|json\|cloudflare`) |
| `botcheck status` | Show loaded sources, data version, and totals |
| `botcheck --version` | Print the version |

## Covered bots

**Search:** Googlebot, Google common crawlers, Bingbot, DuckDuckBot, Applebot ·
**Fetch:** Google user-triggered fetchers ·
**AI:** ChatGPT-User, OpenAI SearchBot, Claude, Perplexity ·
**SEO / Social:** Ahrefs, Facebook/Meta

Run `botcheck status` for live counts per source.

## Custom sources

Create `botcheck.yaml` to add your own lists or pin the registry:

```yaml
cache_dir: "/tmp/botcheck-cache"
registry_ref: "main"   # pin registry to a tag/SHA, or "off" to disable it

sources:
  - name: my-list
    category: monitoring
    type: http                       # JSON feed or plain-text CIDR list
    url: https://example.com/bots.txt
    enabled: true

  - name: internal
    category: monitoring
    type: file                        # local file, one CIDR/IP per line
    path: ./internal-ranges.txt
    enabled: true
```

Your entries override registry ones with the same `name`. Large setups can
split across files with `imports: ["sources.d/*.yaml"]`.

## How it works

1. `update` fetches the registry manifest plus each feed, and builds a
   longest-prefix-match trie cached under `~/.cache/botcheck/`.
2. `scan`/`check` parse IPs and look them up locally — no network, no API keys.
3. Offline with a populated cache, `update` keeps last-good data instead of failing.

## License

Apache-2.0
