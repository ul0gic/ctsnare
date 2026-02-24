# ctsnare — Changelog

> **Document Location:** `.project/changelog.md`
>
> All notable changes to this project will be documented in this file.
> Format based on [Keep a Changelog](https://keepachangelog.com/).

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
| 0.3.0 | Integration Complete | TBD |
| 0.2.0 | Core Engine Complete | TBD |
| 0.1.0 | Foundation Complete | 2026-02-24 |

---

*Last updated: 2026-02-24*
