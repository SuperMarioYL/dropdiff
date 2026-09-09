# Changelog

All notable changes to DropDiff are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0] — 2026-09-09

Post-ship grill iteration: two correctness fixes to the m1 parser and the m3
differ, plus the lockstep version bump.

### Fixed — m1 wired-config detection
- `scanKeyValue` claimed to strip a leading `- ` list bullet, but the
  hyphen-to-underscore key normalization ran first, so `- MODEL=glm-4.6`
  parsed as key `_ MODEL` and was silently dropped. `dropdiff init` now
  captures model/base_url/context hints inlined as `- ` / `* ` list bullets
  in `./CLAUDE.md`, `.env` fragments, and aider conf YAML lists.

### Added — m3 diff coverage
- When a successor drop switches pricing unit (e.g. CNY/M → USD/M),
  `dropdiff check` now prints one amber informational `pricing.unit` delta
  instead of silently emitting no pricing deltas at all. Numeric prices are
  still never compared across units.

### Changed
- Version 0.1.0 → 0.2.0 lockstep across every surface: `VERSION`,
  `cmd/root.go`, tests, README badges, `web/site.json`
  (`meta.content_version` + `footer.tag`), `docs/demo-results.json`, and the
  recorded `assets/demo.cast` `--version` output.

## [0.1.0] — 2026-08-09

First release. The Renovate/Dependabot spec-diff primitive, pointed at the
国产模型 carousel (DeepSeek / Qwen / Kimi / GLM).

### Added — m1: wired config
- `dropdiff init` detects `~/.claude.json` (or `./CLAUDE.md`),
  `~/.aider/conf.yml`, and `.env` and snapshots model + base_url + context +
  cache_mode to `~/.dropdiff/wired.yaml`.

### Added — m2: drop-spec registry
- Curated `data/registry.yaml` covering 8 weeks of DeepSeek/Qwen/Kimi/GLM
  drops with pricing/context/cache/quant/tool-call fields, every row sourced
  from a ModelScope card URL + a 国产 SDK commit SHA.
- `dropdiff sync` pulls the hosted raw-GitHub registry refresh to
  `~/.dropdiff/registry.yaml` (`$DROPDIFF_REGISTRY_URL` overrides).
- `dropdiff list` prints the registry with sources (≥8 drops).
- `dropdiff check` falls back to the **bundled** registry (embedded in the
  binary) before the first sync, so the happy path is zero-network.

### Added — m3: diff + check
- `dropdiff check` diffs the registry against `wired.yaml` and prints only
  config-affecting deltas in red/green via lipgloss (pricing, context, cache
  price, available quants, tool-call quirks). The `Affects` boolean — "does
  this delta touch a field you actually wired?" — is the moat.
- `assets/demo.cast` records the GLM-5.3 cache-price catch (0.28 → 0.14
  CNY/M, green, affects your wired GLM-4.6) before the ModelScope card updates.

### Tooling
- GoReleaser release workflow (`linux/darwin/windows × amd64/arm64`,
  `.tar.gz` + sha256 `checksums.txt`).
- CI workflow (`go test ./...` + `go vet`).
- vhs demo workflow re-renders `assets/demo.gif` from `docs/demo.tape`.
- House-style animated `hero`/`atlas` dark+light SVG pairs + bilingual
  README (zh primary, `README.en.md` sibling).
