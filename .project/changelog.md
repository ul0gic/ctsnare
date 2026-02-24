# ctsnare — Changelog

> **Document Location:** `.project/changelog.md`
>
> All notable changes to this project will be documented in this file.
> Format based on [Keep a Changelog](https://keepachangelog.com/).

---

## [Unreleased]

### Added
- (Phase 2 work pending)

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
