# botcheck

Find known bots in your logs. `botcheck` matches visitor IPs against a
curated registry of verified crawler ranges — Googlebot, Bingbot, GPTBot,
Applebot, Meta and more — at ~1M log lines per second.

## Quick start

```bash
go install github.com/saurabhsharma2u/botcheck/cmd/botcheck@latest

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
| `botcheck check 40.77.167.61` | Check a single IP (`--json` for machine output, `--dns` to confirm via reverse DNS) |
| `botcheck scan access.log` | Check every IP in a log file (`--only-matched`, `--output text\|json\|csv`, `--input nginx|apache|json|cloudflare`) |
| `botcheck status` | Show loaded sources, data version, and totals |
| `botcheck --version` | Print the version |

## Covered bots

**Search:** Googlebot, Google common crawlers, Bingbot, DuckDuckBot, Applebot, Yandex ·
**Fetch:** Google user-triggered fetchers ·
**AI:** ChatGPT-User, OpenAI SearchBot, Claude, PerplexityBot, Perplexity-User ·
**SEO / Social:** Ahrefs, Facebook/Meta

Run `botcheck status` for live counts per source.

## Verifying crawlers

A registry match proves network origin; `--dns` additionally confirms
identity with forward-confirmed reverse DNS (PTR hostname must sit under
the source's documented domain *and* resolve back to the IP):

```bash
botcheck check 66.249.66.1 --dns
# MATCH ... DNS: verified (crawl-66-249-66-1.googlebot.com)
```

Covered: Google (`googlebot.com`, fetchers `googleusercontent.com`),
Bing (`search.msn.com`), Yandex (`yandex.ru/net/com`), Apple
(`applebot.apple.com`). Sources without documented domains report
`unverifiable` — never spoof. DNS runs only on matches and never flips
one; a mismatch is reported, not hidden. Requires network.

No range match? `--dns` falls back automatically: the PTR hostname is
checked against *every* known source's domains and forward-confirmed.
A genuine bot from an unlisted range (Yandex rotates undisclosed IPs)
reports `verified via <source>`; anything else shows the PTR as a
triage hint. One pass only — each hostname resolved once, results never
re-enter lookup, so no loops.

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

## Scheduled refresh

Units ship inside the binary — no extra files needed whatever install
method you used. This installs a daily run (systemd user timer on Linux,
LaunchAgent on macOS):

```bash
botcheck schedule install
botcheck schedule status
botcheck schedule uninstall
```

Options: `--at 04:30` sets the daily time, `--unit-config PATH` bakes a
config file into the unit (default: normal config discovery).
`refresh` takes an overlap lock (`<cache>/refresh.lock`), so overlapping
scheduled runs can never corrupt the cache.

## How it works

1. `update` fetches the registry manifest plus each feed, and builds a
   longest-prefix-match trie cached under `~/.cache/botcheck/`.
2. `scan`/`check` parse IPs and look them up locally — no network, no API keys.
3. Offline with a populated cache, `update` keeps last-good data instead of failing.

## License

Apache-2.0
