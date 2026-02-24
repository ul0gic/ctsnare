# ctsnare — Changelog

> **Document Location:** `.project/changelog.md`
>
> All notable changes to this project will be documented in this file.
> Format based on [Keep a Changelog](https://keepachangelog.com/).

---

## [Unreleased] — Phase 4 Hardening

### Testing — QA (qa-engineer) Phase 4.1 Test Coverage Expansion
- Exhaustive table-driven heuristics tests: 53 test cases covering matchKeywords, scoreTLD, scoreDomainLength, scoreHyphenDensity, scoreNumberSequences, scoreMultiKeywordBonus, registeredPart with edge cases (empty input, nil slices, unicode, case variations, boundary values)
- Manager lifecycle tests with httptest mock CT log server: start/stop, context cancellation stops all pollers, multiple log configs verified via HTTP request counting, empty config, stop-before-start safety
- Config defaults validation: CT log URLs are valid HTTPS, batch size positive, poll interval positive, DB path non-empty, XDG path construction with/without XDG_DATA_HOME, applyDefaults fills zeros while preserving existing values
- Storage edge cases: concurrent reads under WAL mode, 253-char domain names, unicode/punycode/cyrillic/CJK domains, empty/nil keyword arrays, empty string fields, all QueryFilter fields set simultaneously, pagination coverage (page1/page2/past-end), sort by every allowed column, duplicate domain insert error, nonexistent session clear, SQL injection attempts on sort column
- Feed model behavior tests: initial state, hit prepend order, buffer max size enforcement (500 cap), stats message updates, view output contains domain/header/status bar, viewport resize, severity styling, keyword count accumulation and sort order, content width narrow/wide, prependHit/updateKeywordCounts helper functions
- Full suite passes: `go test -v -count=1 -race ./...` -- zero failures, zero data races
- Coverage: scoring 100%, profile 100%, config 95.1%, storage 82.0%, overall 63.6%

### Security — SEC (security-engineer) Phase 4.2 Security Audit
- Security audit v1 complete: audited internal/storage/ (SQL injection), internal/poller/ (SSRF, input validation), internal/config/ (path traversal, DoS)
- govulncheck: zero known vulnerabilities in dependencies
- go mod verify: all modules verified, no integrity issues
- 6 findings documented in .project/issues/open/ISSUE-001-security-audit-v1.md (1 HIGH, 2 MEDIUM, 3 LOW/informational)
- FINDING-01 (MEDIUM): Unbounded HTTP response body read in poller — recommend io.LimitReader
- FINDING-02 (MEDIUM): HTTP client follows redirects without restriction — recommend disabling redirects
- FINDING-03 (LOW): fmt.Sprintf for ORDER BY clause — safe due to allowlist, recommend adding safety comment
- FINDING-04 (LOW): No config file size limit — informational for CLI tool
- FINDING-05 (LOW): DB path from config not validated — informational for CLI tool
- FINDING-06 (HIGH): Missing PRAGMA busy_timeout causes silent data loss under concurrent writes — see ISSUE-002
- Filed ISSUE-002-sqlite-busy-timeout-missing.md for the missing busy_timeout PRAGMA
- SQL injection defenses verified: all queries use parameterized placeholders, sort column allowlisted

### Added — OPS (devops-engineer) Phase 4.3 CI/CD & Build Infrastructure
- Makefile with build, test, lint, fmt, vet, clean, coverage, check, run, help targets
- GitHub Actions CI workflow: lint (golangci-lint-action), test (race detection), build (artifact upload) on push/PR to main
- GitHub Actions release workflow: GoReleaser on tag push (v*) with GitHub release creation
- GoReleaser config: cross-compile linux/darwin/windows amd64/arm64, tar.gz + zip archives, checksums

---

## [0.3.0] - 2026-02-24

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

## [Unreleased]

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

## [0.1.0] - 2026-02-24

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

### Phase
- Phase 1: Foundation — COMPLETE

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
| 0.5.0 | Feature Complete | TBD |
| 0.3.0 | Integration Complete | 2026-02-24 |
| 0.2.0 | Core Engine Complete | TBD |
| 0.1.0 | Foundation Complete | 2026-02-24 |

---

*Last updated: 2026-02-24*
