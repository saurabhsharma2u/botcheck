# Registry Plan — runtime source registry (same repo)

## Goal

Replace the compile-time embedded snapshot (`defaults/`, `DefaultConfig`,
`embed:` paths) with a runtime-fetched registry living in this repo, so
adding or refreshing a source never requires a binary release. Every
`botcheck refresh` fetches fresh data; binary tags stay code-only.

## Layout

```
registry/
  manifest.yaml          # index of every source (schema below)
  meta/fb.txt            # Meta/Facebook ranges (no publisher feed exists)
```

Provider folders hold one file per feed (`google/googlebot.json` …) only
for lists with no upstream. Everything with a publisher URL stays a pointer
in the manifest — no mirroring.

## Manifest schema (`registry/manifest.yaml`)

```yaml
version: "2026-09-11"    # data date, bump on any change
sources:
  - name: googlebot
    category: search
    type: http
    url: https://developers.google.com/.../googlebot.json
    enabled: true
  - name: facebook
    category: social
    type: http
    url: https://raw.githubusercontent.com/<org>/iambot/main/registry/meta/fb.txt
    enabled: true
```

`fb.txt` format: one CIDR or bare IP per line, `#` comment lines allowed,
header stamps origin (`radb.net`, `stat.ripe.net`) + extraction date.
Overlaps kept; longest-prefix match resolves them.

## Migration from `defaults/`

1. Move the verified sources from `defaults/default-sources.yaml` into
   `registry/manifest.yaml` as `http` pointers (same name/category/url),
   plus the `facebook` entry serving `meta/fb.txt` as plain text.
2. Extract the `ip` values from the Facebook JSON into `registry/meta/fb.txt`
   (dedupe exact duplicates, keep overlaps), add provenance header.
3. Add the `facebook` entry pointing at the raw GitHub URL of `fb.txt`.

## Client changes

- `config`: add `registry_url` (default: raw GitHub URL of
  `registry/manifest.yaml` on `main`) and `registry_ref` (branch/tag/SHA,
  default `main`). `registry_url: "off"` = offline mode (local sources only).
- `refresh`: fetch manifest → build source list → merge with `botcheck.yaml`
  sources and `imports:` (local entries win on duplicate `name`, same rule
  as today) → fetch each → save. Manifest fetch failure with a populated
  disk cache = warn + keep last-good; with an empty cache = hard error.
- `status`: show registry ref + data age so staleness is visible.
- Remove: `defaults/` package + dir, `config.DefaultConfig`, the `embed:`
  path scheme in `source/file.go`, zero-config embedded fallback in
  `update.go`. `type: file` stays for user-local custom lists.
- `CODEOWNERS`: `registry/` reviewable by list-maintainers without Go review.

## Test plan (all implemented)

- Manifest parse + merge-order test (registry underlays, local wins).
- `httptest` refresh flow: manifest + good feed + failing feed + local
  override (`TestRunUpdateRegistryMerge`).
- Offline with populated cache keeps last-good, exit 0
  (`TestRunUpdateOfflineKeepsLastGood`).
- Offline with empty cache is a hard error
  (`TestRunUpdateOfflineEmptyCacheErrors`).
- Registry disable switch (`TestRunUpdateRegistryOff`).
- `fb.txt` loader test: every line parses as CIDR or bare IP, no dupes.
- Full suite (`go test -race ./...`), `go vet`, `golangci-lint`, `gofmt`.

## Non-goals (later)

- Separate `iambot-registry` repo — switchover is a one-line default-URL
  change + file move once data PRs dominate (see discussion).
- `sha256` pinning per manifest entry (warn on mismatch).
- ETag/`If-Modified-Since` polite refetch.
- Per-source refresh intervals.

## Order of work (implemented)

1. `registry/manifest.yaml` (12 entries: 11 pointers + facebook) + `registry/meta/fb.txt`.
2. Client: config knobs → manifest fetch + merge → remove embed logic.
3. `status` age display + CODEOWNERS.
4. Tests + docs (README points at `registry/`, not `defaults/`).
