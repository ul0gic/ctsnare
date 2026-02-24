# ctsnare — Changelog

> **Document Location:** `.project/changelog.md`
>
> All notable changes to this project will be documented in this file.
> Format based on [Keep a Changelog](https://keepachangelog.com/).

---

## [0.6.2-alpha] - 2026-02-24 — Enhancement Integration (Phase 7.3)

### Added — CLI (cli-engineer) Phase 7.3: Enhancement Integration

#### Wire Enrichment into Watch Command (7.3.1)
- Bridged enrichment result channel to TUI via `waitForEnrichment` tea.Cmd that converts `EnrichResult` to `EnrichmentMsg`
- Added `discardChan` (buffered 256) to poller `Manager.Start` and `Poller` — zero-scored domains are sent on it for TUI activity feed display
- `AppModel` now receives and manages `enrichChan` and `discardChan` — both wired in `Init()` with re-subscription on each message
- `EnrichmentMsg` handler updates matching hits in feed, explorer, and detail views with live enrichment data (IsLive, ResolvedIPs, HostingProvider, HTTPStatus)
- `DiscardedDomainMsg` handler re-subscribes to discard channel after each message
- Headless mode now starts enrichment pipeline — creates enricher, enqueues domains from hitChan, drains enrichment results and discards silently

#### End-to-End Integration Tests (7.3.2)
- Bookmark workflow: insert hits, bookmark via store, query with `--bookmarked`, verify only bookmarked hit returned
- Delete workflow: single delete and batch delete verified through query absence
- Enrichment fields in JSON output: `IsLive`, `ResolvedIPs`, `HostingProvider`, `HTTPStatus` verified present after `UpdateEnrichment`
- `--live-only` flag: returns only enriched live domains
- `--bookmarked` + `--live-only` composition: both flags AND correctly
- `db export --format jsonl` includes enrichment fields
- `db export --format csv` includes enrichment column headers (`is_live`, `resolved_ips`, `hosting_provider`, `http_status`, `bookmarked`, `live_checked_at`)
- `resetFlags()` now covers `queryBookmarked` and `queryLiveOnly` to prevent test leaks

### Changed
- `tui.NewApp` signature expanded to accept `enrichChan` and `discardChan` parameters
- `poller.Manager.Start` signature expanded with `discardChan chan<- string` parameter
- `poller.NewPoller` signature expanded with `discardChan chan<- string` parameter

---

## [0.6.0-alpha] - 2026-02-24 — Enhancement Foundation (Phase 7.1)

### Added — BE (backend-engineer) Phase 7.1: Enhancement Foundation

#### Domain Type Extensions (7.1.1)
- Extended `Hit` struct with enrichment fields: `IsLive`, `ResolvedIPs`, `HostingProvider`, `HTTPStatus`, `LiveCheckedAt`, `Bookmarked` — all zero-value safe
- Extended `QueryFilter` with `Bookmarked` and `LiveOnly` filter fields
- Extended `Store` interface with `SetBookmark`, `DeleteHit`, `DeleteHits`, `UpdateEnrichment` methods

#### Storage Schema Migration (7.1.2)
- Idempotent V2 schema migration adding 6 new columns to the `hits` table with partial indexes on `bookmarked` and `is_live`
- Full CRUD for bookmark toggle, single/batch delete, and enrichment updates
- Extended `UpsertHit`, `InsertHit`, `QueryHits`, and `scanHit` for all new fields
- `sanitizeSortColumn` now accepts `is_live`, `bookmarked`, `http_status`, `live_checked_at`
- 12 new storage tests covering migration idempotency, bookmark CRUD, delete single/batch, enrichment roundtrip, and new query filters

#### TUI Message Types & Shared Contracts (7.1.3)
- New Bubble Tea message types: `EnrichmentMsg`, `BookmarkToggleMsg`, `DeleteHitsMsg`, `DiscardedDomainMsg`
- Added `HitsPerMin` field to `PollStats`
- New Lipgloss styles: `StyleLiveDomain`, `StyleDiscardedDomain`, `StyleBookmarked`, `StyleSelectedCheckbox` with adaptive color constants
- New key bindings: Bookmark (b), Delete (d), SelectToggle (space), SelectAll (a), DeselectAll (A), ConfirmDelete (D)

#### Enrichment Package Scaffold (7.1.4)
- New `internal/enrichment/` package with rate-limited worker pool (5 workers, 5 req/sec global rate via `x/time/rate`)
- DNS resolver with CIDR range matching for Cloudflare, Fastly, Akamai, DigitalOcean and reverse DNS patterns for AWS, GCP, Azure
- HTTP liveness probe with HTTPS-first, HTTP-fallback, HEAD-only, 3-redirect limit, 5s timeout
- Added `golang.org/x/time` dependency

---

## [0.5.0] - 2026-02-24 — Feature Complete

### Added — DOC (documentation-engineer) Phase 6: Documentation

- `README.md` at project root: full subcommand reference with copy-pasteable examples for `watch`, `query`, `db`, and `profiles`; configuration section covering TOML config file, all options with defaults, and custom profile definition; built-in profile table (crypto, phishing, all); scoring heuristics explanation (six heuristics, severity thresholds); architecture data flow diagram (ASCII); development setup (prerequisites, build, test, `make check`)
- Go doc comments expanded across all packages with field-level documentation on exported structs (`Hit`, `QueryFilter`, `DBStats`, `Profile`, `CTLogEntry`, `ScoredDomain`, `PollStats`, `CTLogConfig`, `Config`) and method-level documentation on all interface methods (`Scorer.Score`, `Store.*`, `ProfileLoader.*`)
- CLI help text improved for all commands: `watch`, `query`, `db`, `db clear`, `db export`, `profiles`, `profiles show` — each now has a Long description with usage examples and flag descriptions that include defaults and accepted values

---

## [0.4.0] - 2026-02-24 — Hardened & CI-Ready

### Security — BE (backend-engineer) Phase 5.1 Security Remediation

- Fixed FINDING-06 / ISSUE-002 (HIGH): Added `PRAGMA busy_timeout=5000` in `internal/storage/db.go` to prevent SQLITE_BUSY data loss under concurrent writes from multiple poller goroutines
- Fixed FINDING-01 (MEDIUM): Added `io.LimitReader` with 50 MB cap on HTTP response bodies in `internal/poller/ctlog.go` to prevent memory exhaustion from oversized CT log responses
- Fixed FINDING-02 (MEDIUM): Disabled HTTP redirect following in CT log client (`CheckRedirect` returns `http.ErrUseLastResponse`) to prevent SSRF via compromised log servers
- Fixed FINDING-03 (LOW): Added security invariant comment to ORDER BY clause construction in `internal/storage/hits.go` documenting allowlist safety
- Resolved ISSUE-001 (security audit) and ISSUE-002 (busy timeout) — moved to closed

### Changed — BE (backend-engineer) Phase 5.2 Internal Documentation

- Updated `/.claude/CLAUDE.md` with final project structure (doc.go files, messages.go, CI/CD artifacts, .goreleaser.yml), `make check` as primary verification command

### Added — QA (qa-engineer) Phase 4.1 Test Coverage Expansion

- Exhaustive table-driven heuristics tests: 53 test cases covering matchKeywords, scoreTLD, scoreDomainLength, scoreHyphenDensity, scoreNumberSequences, scoreMultiKeywordBonus, registeredPart with edge cases (empty input, nil slices, unicode, case variations, boundary values)
- Manager lifecycle tests with httptest mock CT log server: start/stop, context cancellation stops all pollers, multiple log configs verified via HTTP request counting, empty config, stop-before-start safety
- Config defaults validation: CT log URLs are valid HTTPS, batch size positive, poll interval positive, DB path non-empty, XDG path construction with/without XDG_DATA_HOME, applyDefaults fills zeros while preserving existing values
- Storage edge cases: concurrent reads under WAL mode, 253-char domain names, unicode/punycode/cyrillic/CJK domains, empty/nil keyword arrays, empty string fields, all QueryFilter fields set simultaneously, pagination coverage (page1/page2/past-end), sort by every allowed column, duplicate domain insert error, nonexistent session clear, SQL injection attempts on sort column
- Feed model behavior tests: initial state, hit prepend order, buffer max size enforcement (500 cap), stats message updates, view output contains domain/header/status bar, viewport resize, severity styling, keyword count accumulation and sort order, content width narrow/wide, prependHit/updateKeywordCounts helper functions
- Full suite passes: `go test -v -count=1 -race ./...` — zero failures, zero data races
- Coverage: scoring 100%, profile 100%, config 95.1%, storage 82.0%, overall 63.6%

### Security — SEC (security-engineer) Phase 4.2 Security Audit

- Security audit v1 complete: audited internal/storage/ (SQL injection), internal/poller/ (SSRF, input validation), internal/config/ (path traversal, DoS)
- govulncheck: zero known vulnerabilities in dependencies
- go mod verify: all modules verified, no integrity issues
- 6 findings documented in .project/issues/open/ISSUE-001-security-audit-v1.md (1 HIGH, 2 MEDIUM, 3 LOW/informational)
- SQL injection defenses verified: all queries use parameterized placeholders, sort column allowlisted

### Added — OPS (devops-engineer) Phase 4.3 CI/CD & Build Infrastructure

- Makefile with build, test, lint, fmt, vet, clean, coverage, check, run, help targets
- GitHub Actions CI workflow: lint (golangci-lint-action), test (race detection), build (artifact upload) on push/PR to main
- GitHub Actions release workflow: GoReleaser on tag push (v*) with GitHub release creation
- GoReleaser config: cross-compile linux/darwin/windows amd64/arm64, tar.gz + zip archives, checksums

---

## [0.3.0] - 2026-02-24 — Integration Complete

### Added — CLI (cli-engineer) Phase 3 Integration

- Watch command wired to real components: config loading, storage, scoring engine, profile manager, poller manager, TUI dashboard with live hit/stats channels, and headless polling mode (internal/cmd/watch.go)
- Poller-to-TUI stats bridge: aggregates per-log PollStats into TUI PollStats with certs/sec calculation and active log count
- Query command wired to real storage: config-based DB path, QueryFilter from flags, table/JSON/CSV output, friendly "no database" message (internal/cmd/query.go)
- DB subcommands wired to real storage: stats reads from SQLite, clear supports --confirm and --session, export writes JSONL/CSV to file or stdout, path reads from config (internal/cmd/db.go)
- Profiles command wired to real profile manager: list shows all profiles with descriptions, show displays keywords/TLDs/skip suffixes (internal/cmd/profiles.go)
- Structured logging via slog: JSON handler at Debug level when --verbose, discarded otherwise to keep TUI clean. PersistentPreRunE in root command configures the global logger (internal/cmd/root.go)
- Signal handling: headless mode uses signal.NotifyContext for SIGINT/SIGTERM, TUI mode cancels context on program exit, both paths drain channels and stop pollers gracefully
- Integration test suite: 14 tests covering query with filters (keyword, severity, session, score-min, limit, format), db stats/clear/export/path, profiles list/show, and root help (internal/cmd/integration_test.go)

---

## [0.2.0] - 2026-02-24 — Core Engine Complete

### Added — BE (backend-engineer) Phase 2 tasks

- Configuration system: TOML config loading via BurntSushi/toml, XDG-compliant paths, sensible defaults, CLI flag merge (internal/config/)
- Keyword profile system: Built-in crypto, phishing, and combined ("all") profiles with curated keyword lists, suspicious TLD sets, and infrastructure skip suffixes. Profile Manager satisfies domain.ProfileLoader with custom profile extension support (internal/profile/)
- Scoring engine: Six heuristics — keyword substring match (2 pts each), suspicious TLD (+1), domain length (+1 if >30 chars), hyphen density (+1 if 2+), digit sequences (+1 if 4+), multi-keyword bonus (+2 if 3+ matches). Severity classification: HIGH >= 6, MED 4-5, LOW 1-3. Skip suffix short-circuit (internal/scoring/)
- SQLite storage layer: Pure Go SQLite via modernc.org/sqlite, WAL mode, parameterized queries, dynamic query builder with keyword/score/severity/session/TLD/time filters, sort column whitelist for SQL injection prevention, upsert domain deduplication, session management, stats aggregation with top keyword counting, JSONL and CSV export (internal/storage/)
- CT log poller: RFC 6962 HTTP client (get-sth, get-entries) with 429 rate limit backoff, MerkleTreeLeaf/TimestampedEntry decoder supporting x509 and pre-certificate entries, domain extraction (CN + SAN DNS names) with deduplication, per-log goroutine polling loop with context-aware graceful shutdown, multi-poller Manager with WaitGroup coordination (internal/poller/)

### Added — CLI (cli-engineer) Phase 2 tasks

- TUI Lipgloss style definitions with adaptive colors for light/dark terminals (styles.go)
- TUI key bindings with vim-style navigation and help bubble integration (keys.go)
- TUI shared message types: HitMsg, StatsMsg, HitsLoadedMsg, SwitchViewMsg, ShowDetailMsg, PollStats (messages.go)
- TUI live feed view with scrollable viewport, severity-colored hit lines, stats bar, and top keywords sidebar (feed.go)
- TUI DB explorer view with sortable bubbles table, filter status bar, and Store-backed query loading (explorer.go)
- TUI detail view with full hit record display including SANs, issuer, keywords, cert info (detail.go)
- TUI filter overlay with text inputs for keyword, score, severity, time range, session (filter.go)
- TUI root app model with view switching (feed/explorer/detail/filter), channel subscriptions, and message routing (app.go)
- TUI app tests covering initialization, tab switching, hit messages, quit, detail/filter overlay (app_test.go)
- Cobra `watch` subcommand with --profile, --session, --headless, --batch-size, --poll-interval flags (watch.go)
- Cobra `query` subcommand with --keyword, --score-min, --since, --tld, --session, --severity, --format, --limit flags (query.go)
- Cobra `db` subcommand with stats, clear, export, path sub-subcommands (db.go)
- Cobra `profiles` subcommand with list and show sub-subcommand (profiles.go)
- Shared output formatters: table (tabwriter), JSON (JSONL), CSV, stats pretty-printer (output.go)

---

## [0.1.0] - 2026-02-24 — Foundation Complete

### Added

- Project scaffold: Go 1.26 module, directory structure, entry point
- Production dependencies: Cobra, Bubbletea, Lipgloss, Bubbles, modernc.org/sqlite, BurntSushi/toml
- Dev dependencies: testify
- Core domain types: Hit, Severity, CTLogEntry, ScoredDomain
- Core interfaces: Scorer, Store, ProfileLoader
- Query types: QueryFilter, DBStats, KeywordCount
- Profile type: Profile struct
- Cobra root command with persistent flags (--config, --db, --verbose)
- golangci-lint configuration (errcheck, govet, staticcheck, unused, gosimple, ineffassign, gofmt, goimports)
- .gitignore for binary, database, and build artifacts

---

## Version Guidelines

### Version Format: `MAJOR.MINOR.PATCH`

- **MAJOR**: Breaking changes or significant milestones
- **MINOR**: New features, completed phases
- **PATCH**: Bug fixes, small improvements

### Change Types

| Type | Description |
|------|-------------|
| **Added** | New features or capabilities |
| **Changed** | Changes to existing functionality |
| **Deprecated** | Features marked for removal |
| **Removed** | Features that were removed |
| **Fixed** | Bug fixes |
| **Security** | Security-related changes |

---

## Milestones

| Version | Milestone | Date |
|---------|-----------|------|
| 1.0.0 | Production Release | TBD |
| 0.5.0 | Feature Complete | 2026-02-24 |
| 0.3.0 | Integration Complete | 2026-02-24 |
| 0.2.0 | Core Engine Complete | 2026-02-24 |
| 0.1.0 | Foundation Complete | 2026-02-24 |

---

*Last updated: 2026-02-24*
