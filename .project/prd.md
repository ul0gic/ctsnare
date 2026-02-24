# [Project Name] - Product Requirements Document

> **Document Location:** `.project/prd.md`
>
> This document defines the product requirements, features, and specifications.
> Keep this document as the single source of truth for what we're building.

---
# cert-hunter — Product Requirements Document

## Overview

**cert-hunter** is a compiled Go CLI tool that monitors Certificate Transparency logs in real-time, scores newly issued domains against configurable keyword profiles, stores hits in an embedded SQLite database, and provides both a live TUI dashboard and an interactive database explorer — all in a single, zero-dependency binary.

The tool fills a gap in the current landscape: existing CT tools are either domain-lookup utilities (give it a domain, get its certs) or raw bulk downloaders. None of them do real-time keyword hunting with scoring heuristics and queryable structured storage. cert-hunter is for fraud analysts, threat intel teams, bug bounty hunters, and anyone who needs an early warning system for suspicious domains going live.

---

## Origin

This project started as a Python proof-of-concept built during live recon on a scam domain. The prototype polls three Google CT logs directly (no third-party relay), scores domains by keyword match density and TLD suspiciousness, and outputs color-coded hits to terminal with JSONL logging. It works — it's catching real hits across crypto, casino, and phishing domains. But the Python version has limitations: no persistent storage, no queryability after the fact, dedup state lives in memory (kill the process, lose it), and distribution requires a Python environment. The Go rewrite solves all of that with a single binary that embeds everything.

---

## Why Go

- **Single binary distribution.** `go install` or download a binary. No runtime, no venv, no pip.
- **Embedded SQLite via pure Go.** `modernc.org/sqlite` compiles SQLite directly into the binary — no CGo, no C compiler, no external library. The database engine ships inside the tool.
- **Concurrency model fits perfectly.** CT log pollers run as goroutines pushing entries through channels. The TUI subscribes to the same channel. Polling never stops when switching views.
- **Cross-compilation is trivial.** Single `GOOS=linux GOARCH=amd64 go build` for any target platform.
- **stdlib covers most needs.** HTTP client, x509 parsing, JSON encoding — no external deps for core functionality.

---

## Core Architecture

### Single Binary, Subcommand Structure

- `cert-hunter watch` — Live monitoring mode (TUI dashboard)
- `cert-hunter watch --headless` — No TUI, poll and store silently (for servers, cron, background runs)
- `cert-hunter query` — Search and filter stored hits from SQLite
- `cert-hunter db` — Database management (clear, stats, export)
- `cert-hunter profiles` — List and inspect keyword profiles

### Decoupled Polling and Display

The CT log pollers are completely independent of the display layer. Pollers run as goroutines, parse certificates, score domains, and write to SQLite continuously. The TUI is a subscriber — it renders whatever the pollers produce but never blocks or controls them. This means:

- Switching from live feed to DB explorer doesn't pause polling
- Headless mode uses the same poller code, just skips the TUI
- The query subcommand bypasses polling entirely and reads directly from SQLite

### Data Flow

```
CT Logs (Google Argon, Xenon, etc.)
        │
        ▼
  Poller Goroutines (one per log)
        │
        ▼
  Scoring Engine (keyword match + heuristics)
        │
        ├──▶ SQLite (persistent storage, always)
        │
        └──▶ TUI Channel (live feed, when watching)
```

---

## Features

### 1. Real-Time CT Log Polling

Direct HTTP polling of Certificate Transparency logs using the RFC 6962 API (`get-sth`, `get-entries`). No websocket relays, no third-party services. Straight from Google's infrastructure.

- Track position per log to resume where we left off
- Configurable batch size (entries fetched per poll per log)
- Configurable poll interval
- Graceful handling of log errors, rate limits, and temporary unavailability
- Automatic discovery and rotation of active CT log shards (2026h1 → 2026h2, etc.)

### 2. Domain Scoring Engine

Every domain extracted from a certificate is scored against the active keyword profile. The score determines severity (HIGH / MED / LOW) and drives filtering and prioritization.

**Scoring heuristics:**

- **Keyword matches** — Base score: 2 points per keyword found in the domain
- **TLD suspiciousness** — +1 for known sketchy TLDs (.xyz, .top, .vip, .win, .bet, .casino, etc.)
- **Domain length** — +1 for domains over 30 characters
- **Hyphen density** — +1 for 2+ hyphens (classic scam domain pattern)
- **Number sequences** — +1 for 4+ consecutive digits
- **Multi-keyword bonus** — +2 for 3+ keyword matches on a single domain

**Severity thresholds:**

- **HIGH** (score ≥ 6) — Almost certainly malicious. Multi-keyword hit on a sketchy TLD.
- **MED** (score 4–5) — Suspicious. Worth investigating.
- **LOW** (score 1–3) — Single keyword match. Noise-prone but logged for completeness.

**Noise filtering:**

Skip list of known infrastructure suffixes (cloudflaressl.com, amazonaws.com, herokuapp.com, etc.) to filter out platform certificate churn that would otherwise flood results.

### 3. Keyword Profiles

Profiles are named collections of keywords and TLD boost lists. Ship with sensible defaults, allow user customization.

**Built-in profiles:**

- **crypto** — Casino, swap, investment, airdrop, fake exchange scams
- **phishing** — Credential harvesting, fake login pages, bank impersonation
- **all** — Combined

**Custom profiles:**

Users can define their own profiles via TOML configuration file. Custom profiles can extend built-in ones or start from scratch. This supports niche use cases — brand protection, industry-specific monitoring, research-focused keyword sets.

### 4. SQLite Storage

Every hit is persisted to an embedded SQLite database. The database is the single source of truth — the TUI reads from it, queries read from it, exports read from it.

**Schema captures:**

- Domain name
- Score and severity classification
- Matched keywords
- Certificate issuer (org + CN)
- All SAN domains from the certificate
- Certificate not-before timestamp
- Which CT log the entry came from
- Which profile was active
- Session tag (for grouping runs)
- Timestamp of when the hit was recorded

**Indexes on:** score (descending), domain, session, created_at for fast filtering and sorting.

**Deduplication:** Domain uniqueness is enforced. If the same domain appears in a later certificate, the existing record is updated (upsert) rather than duplicated. This replaces the in-memory `seen` set from the Python version with persistent, crash-safe dedup.

### 5. TUI Dashboard (Watch Mode)

The primary interface. Built with bubbletea + lipgloss (Charm ecosystem). Two views, toggled with a keypress.

**Live Feed View:**

- Scrollable stream of hits as they arrive in real-time
- Each entry shows: timestamp, severity tag (color-coded), domain, score, matched keywords, issuer
- Stats bar at the bottom: total certs scanned, hit count, rate (certs/sec), active profile
- Top keywords sidebar: live-updating frequency chart of which keywords are hitting most

**DB Explorer View:**

- Scrollable table of all hits stored in SQLite
- Sortable by any column: score, domain, timestamp, keyword count
- Filterable: by keyword search, minimum score, severity level, time range, session
- Drill-down: select a row to see full detail — all SAN domains, complete issuer info, raw timestamps, which CT log
- Clear database option with confirmation prompt
- Export current filtered view

**Keybindings:**

- `Tab` — Toggle between Live Feed and DB Explorer
- `/` — Search (vim-style)
- `s` — Cycle sort column
- `f` — Filter menu
- `Enter` — Drill into selected record (Explorer) 
- `C` — Clear database (with confirmation)
- `Esc` — Back / dismiss
- `q` — Quit

**Critical behavior:** Switching views never stops polling. The watcher goroutines are always running. The TUI only changes what gets rendered.

### 6. Query Mode (CLI)

For non-interactive use, scripting, and piping into other tools.

- `cert-hunter query --keyword casino` — All hits containing "casino"
- `cert-hunter query --score-min 5` — Only MED and HIGH severity
- `cert-hunter query --since 24h` — Last 24 hours
- `cert-hunter query --tld .xyz` — Filter by TLD
- `cert-hunter query --session midnight-run` — Filter by session tag
- `cert-hunter query --severity HIGH` — Only high-confidence hits
- `cert-hunter query --format json` — JSON output for piping
- `cert-hunter query --format csv` — CSV export
- `cert-hunter query --format table` — Pretty-printed table (default)

Flags are composable: `cert-hunter query --keyword casino --score-min 4 --since 12h --format json`

### 7. Database Management

- `cert-hunter db stats` — Total hits, breakdown by severity, top keywords, date range
- `cert-hunter db clear` — Nuke everything (with --confirm flag)
- `cert-hunter db clear --session midnight-run` — Clear only a specific session
- `cert-hunter db export --format jsonl` — Full database export
- `cert-hunter db path` — Print the database file path

### 8. Sessions

Sessions are optional tags applied to a monitoring run. They allow grouping and isolating results without destroying data.

- `cert-hunter watch --session midnight-run` — Tag all hits from this run
- `cert-hunter query --session midnight-run` — Query only that session
- `cert-hunter db clear --session midnight-run` — Clear only that session
- No session flag = default session (all hits land in the same bucket)

This eliminates the need to clear the database just to get a fresh view. Run a session, analyze it, start a new one.

---

## Configuration

### Config File (Optional)

TOML file at `~/.config/cert-hunter/config.toml` or specified via `--config` flag. Everything has sensible defaults — the tool works with zero configuration.

**Configurable values:**

- CT log URLs (add/remove logs)
- Default profile
- Batch size and poll interval
- Database path
- Custom keyword profiles
- Custom skip suffixes (noise filter)

### CLI Flags Override Config

Every config file value can be overridden by a CLI flag. CLI flags always win.

---

## Distribution

- `go install github.com/user/cert-hunter@latest` — Primary install method
- Pre-built binaries for Linux (amd64, arm64), macOS (amd64, arm64), Windows (amd64) via GitHub Releases
- Single binary, no runtime dependencies, no setup steps
- Database file auto-created on first run at `~/.local/share/cert-hunter/cert-hunter.db` (XDG-compliant, overridable)

---

## Non-Goals (v1)

- **Webhook / alerting integrations.** Valuable but not v1. Query mode + cron + a shell script covers most alerting needs initially.
- **Web UI.** The TUI is the interface. A web dashboard is a different product.
- **Historical backfill.** The tool starts from the current tree head and watches forward. Backfilling millions of existing entries is a different workflow.
- **WHOIS / DNS enrichment.** Tempting but adds external dependencies and latency. Can be a plugin or post-processing step.
- **Multi-user / auth.** Single-user CLI tool. The database is a local file.

---

## Success Criteria

- Single binary installs and runs on Linux/macOS/Windows with zero setup
- Catches the same hits as the Python prototype (functional parity)
- TUI renders smoothly at sustained throughput (~300+ certs/sec across 3 logs)
- View switching is instant (no polling interruption)
- DB Explorer handles 100k+ rows without lag
- Query mode returns results in under 1 second for typical filters
- Database survives crashes (SQLite WAL mode, no data loss)

---

## Future Possibilities

- **Alerting:** Slack, Discord, webhook push for HIGH severity hits
- **Profile sharing:** Import/export profile configs for team use
- **Enrichment plugins:** WHOIS lookup, DNS resolution, VirusTotal check as optional post-processing
- **Dashboard mode:** Persistent TUI that auto-reconnects and resumes on reboot (systemd service + TUI attach)
- **Pattern matching beyond keywords:** Regex support, Levenshtein distance for brand typosquatting detection
- **Multi-database:** Separate DBs per engagement or client
## Overview

### Problem Statement
[What problem does this solve? Why does it need to exist?]

### Solution
[High-level description of the solution]

### Target Users
- **Primary:** [Who is the main user?]
- **Secondary:** [Other users who benefit]

### Success Metrics
- [ ] [Measurable outcome 1]
- [ ] [Measurable outcome 2]
- [ ] [Measurable outcome 3]

---

## Features

### Core Features (MVP)

#### Feature 1: [Name]
**Priority:** P0 (Must Have)

**Description:**
[What does this feature do?]

**User Story:**
> As a [user type], I want to [action] so that [benefit].

**Acceptance Criteria:**
- [ ] [Criteria 1]
- [ ] [Criteria 2]
- [ ] [Criteria 3]

**Technical Notes:**
- [Any implementation considerations]

---

#### Feature 2: [Name]
**Priority:** P0 (Must Have)

**Description:**
[What does this feature do?]

**User Story:**
> As a [user type], I want to [action] so that [benefit].

**Acceptance Criteria:**
- [ ] [Criteria 1]
- [ ] [Criteria 2]

---

### Secondary Features (Post-MVP)

#### Feature 3: [Name]
**Priority:** P1 (Should Have)

**Description:**
[What does this feature do?]

---

#### Feature 4: [Name]
**Priority:** P2 (Nice to Have)

**Description:**
[What does this feature do?]

---

## User Interface

### Screens/Views

#### Screen 1: [Name]
**Purpose:** [What does this screen do?]

**Components:**
- [Component 1]
- [Component 2]

**User Actions:**
- [Action 1] -> [Result]
- [Action 2] -> [Result]

---

### Design Guidelines

#### Color Palette
| Name | Hex | Usage |
|------|-----|-------|
| Primary | #000000 | Main actions |
| Secondary | #666666 | Supporting elements |
| Accent | #00D9FF | Highlights |
| Background | #0A0A0A | App background |
| Surface | #111111 | Cards/panels |

#### Typography
- **Headings:** [Font family, weights]
- **Body:** [Font family, weights]
- **Code:** [Monospace font]

---

## Technical Requirements

### Platform
- [Platform/OS requirements]
- [Minimum version requirements]

### Performance
- [Load time requirements]
- [Response time requirements]
- [Memory/resource constraints]

### Security
- [Authentication requirements]
- [Data protection requirements]
- [Compliance requirements]

### Data
- [Data storage requirements]
- [Data retention policies]
- [Export/import capabilities]

---

## Constraints & Assumptions

### Constraints
- [Technical limitations]
- [Budget/resource constraints]
- [Timeline constraints]

### Assumptions
- [Assumption 1]
- [Assumption 2]

### Out of Scope
- [Explicitly not included 1]
- [Explicitly not included 2]

---

## Glossary

| Term | Definition |
|------|------------|
| [Term 1] | [Definition] |
| [Term 2] | [Definition] |

---

## Revision History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | YYYY-MM-DD | [Name] | Initial draft |

---

*Last updated: YYYY-MM-DD*
