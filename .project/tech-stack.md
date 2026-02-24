# cert-hunter — Tech Stack

> **Document Location:** `.project/tech-stack.md`
>
> This document outlines the technology choices and rationale for the project.
> All technology decisions should be documented here with reasoning.

---

## Stack Overview

```
┌─────────────────────────────────────────────────┐
│                     CLI / TUI                    │
│  Cobra + Bubbletea + Lipgloss                   │
├─────────────────────────────────────────────────┤
│                  Core Engine                     │
│  Go 1.26 + stdlib net/http + crypto/x509        │
├─────────────────────────────────────────────────┤
│                   Data Layer                     │
│  SQLite (embedded) via modernc.org/sqlite        │
├─────────────────────────────────────────────────┤
│                  Distribution                    │
│  Single binary + GitHub Releases + go install    │
└─────────────────────────────────────────────────┘
```

---

## Core Technologies

### Language & Runtime

| Technology | Version | Purpose |
|------------|---------|---------|
| Go | 1.26.0 | Primary language — compiled, single binary output |

**Rationale:**
- Single binary distribution with zero runtime dependencies
- Embedded SQLite via pure Go (`modernc.org/sqlite`) — no CGo, no C compiler required
- Goroutines and channels are a natural fit for concurrent CT log polling + TUI rendering
- Trivial cross-compilation (`GOOS/GOARCH`)
- stdlib covers HTTP client, x509 parsing, JSON encoding — minimal external deps for core functionality

---

### CLI Framework

| Technology | Version | Purpose |
|------------|---------|---------|
| `github.com/spf13/cobra` | latest | Subcommand CLI structure (`watch`, `query`, `db`, `profiles`) |

**Rationale:**
- Industry standard for Go CLIs with subcommand trees
- Built-in help generation, flag parsing, shell completions
- Alternatives considered: `urfave/cli` (less structured), stdlib `flag` (no subcommand support)

---

### TUI Framework

| Technology | Version | Purpose |
|------------|---------|---------|
| `github.com/charmbracelet/bubbletea` | latest | TUI application framework (Elm architecture) |
| `github.com/charmbracelet/lipgloss` | latest | TUI styling — colors, borders, layout |
| `github.com/charmbracelet/bubbles` | latest | Pre-built TUI components (tables, text inputs, spinners) |

**Rationale:**
- Bubbletea's message-passing model decouples rendering from polling goroutines cleanly
- Lipgloss gives terminal-native styling without ANSI escape code management
- Bubbles provides table, viewport, and text input components out of the box
- Charm ecosystem is the most actively maintained Go TUI stack

---

### Database

| Technology | Version | Purpose |
|------------|---------|---------|
| SQLite | 3.x (embedded) | Persistent storage for all hits |
| `modernc.org/sqlite` | latest | Pure Go SQLite driver — compiles into the binary, no CGo |

**Rationale:**
- Embedded database = no external process, no setup, ships inside the binary
- WAL mode for crash safety and concurrent read/write
- Pure Go driver means cross-compilation works without a C toolchain
- Handles 100k+ rows with proper indexing for the query/explorer use case
- Alternatives considered: Bolt/bbolt (no SQL, harder to query), BadgerDB (overkill for structured data)

**Schema Location:** `internal/storage/schema.go`

---

### Configuration

| Technology | Version | Purpose |
|------------|---------|---------|
| `github.com/BurntSushi/toml` | latest | Parse TOML config files for profiles and settings |

**Rationale:**
- TOML is human-friendly for keyword profiles and config — better than JSON for hand-editing
- BurntSushi/toml is the original Go TOML parser, battle-tested, minimal
- Alternative considered: `pelletier/go-toml` (larger API surface, not needed here)

---

## Dependencies

### Production Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/spf13/cobra` | CLI subcommand framework |
| `github.com/charmbracelet/bubbletea` | TUI application framework |
| `github.com/charmbracelet/lipgloss` | TUI styling and layout |
| `github.com/charmbracelet/bubbles` | TUI components (table, viewport, text input) |
| `modernc.org/sqlite` | Pure Go embedded SQLite driver |
| `github.com/BurntSushi/toml` | TOML config file parsing |
| `log/slog` (stdlib) | Structured logging |

### Development Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/stretchr/testify` | Test assertions and mocking |
| `golangci-lint` | Linter aggregator |

---

## Build & Tooling

### Build System

| Tool | Version | Purpose |
|------|---------|---------|
| `go build` | Go 1.26 | Compile single binary |
| `go test` | Go 1.26 | Run tests |
| Goreleaser (future) | — | Cross-compiled release builds for GitHub Releases |

### Development Tools

| Tool | Purpose |
|------|---------|
| `golangci-lint` | Lint aggregator (staticcheck, errcheck, govet, etc.) |
| `gofmt` / `goimports` | Code formatting |
| `go vet` | Static analysis (included in golangci-lint) |

### Build Commands

```bash
# Development (run directly)
go run ./cmd/cert-hunter

# Production build
go build -o cert-hunter ./cmd/cert-hunter

# Testing
go test ./...

# Linting
golangci-lint run ./...

# Formatting
gofmt -w .
```

---

## Architecture Patterns

### Code Organization

```
cert-hunter/
├── .project/                # Project documentation
│   ├── prd.md
│   ├── tech-stack.md
│   ├── build-plan.md
│   └── changelog.md
├── cmd/
│   └── cert-hunter/
│       └── main.go          # Entry point — Cobra root command setup
├── internal/
│   ├── cmd/                 # Cobra subcommand definitions
│   │   ├── watch.go         # watch command (TUI + headless)
│   │   ├── query.go         # query command (CLI search)
│   │   ├── db.go            # db management commands
│   │   └── profiles.go      # profile listing/inspection
│   ├── poller/              # CT log polling engine
│   │   ├── poller.go        # Per-log goroutine polling loop
│   │   └── ctlog.go         # RFC 6962 API client (get-sth, get-entries)
│   ├── scoring/             # Domain scoring engine
│   │   ├── scorer.go        # Score calculation + severity classification
│   │   └── heuristics.go    # Individual scoring heuristics
│   ├── profile/             # Keyword profile management
│   │   ├── profile.go       # Profile types and loading
│   │   └── builtin.go       # Built-in profiles (crypto, phishing, all)
│   ├── storage/             # SQLite data layer
│   │   ├── db.go            # Database init, migrations, connection
│   │   ├── schema.go        # Table definitions and indexes
│   │   ├── hits.go          # Hit CRUD — insert, upsert, query
│   │   └── sessions.go      # Session management
│   ├── tui/                 # Bubbletea TUI
│   │   ├── app.go           # Root model — view switching, key routing
│   │   ├── feed.go          # Live feed view
│   │   ├── explorer.go      # DB explorer view
│   │   ├── detail.go        # Record drill-down view
│   │   └── styles.go        # Lipgloss style definitions
│   └── config/              # Configuration loading
│       └── config.go        # TOML parsing, defaults, flag merging
├── go.mod
├── go.sum
└── .golangci.yml            # Linter configuration
```

### Design Patterns Used

| Pattern | Where Used | Purpose |
|---------|------------|---------|
| Fan-out/fan-in | `internal/poller/` | Multiple goroutines poll CT logs concurrently, push hits through a shared channel |
| Elm Architecture (TEA) | `internal/tui/` | Bubbletea's Model-Update-View pattern for TUI state management |
| Repository pattern | `internal/storage/` | Database access abstracted behind query methods |
| Strategy pattern | `internal/scoring/` | Pluggable scoring heuristics applied to each domain |
| Config cascade | `internal/config/` | Defaults → config file → CLI flags (flags always win) |

---

## Environment Configuration

### Required Environment Variables

None. cert-hunter is a zero-config tool. Everything is configured via CLI flags or optional TOML config file.

### Configuration Files

| File | Purpose |
|------|---------|
| `~/.config/cert-hunter/config.toml` | Optional user configuration (profiles, log URLs, defaults) |
| `~/.local/share/cert-hunter/cert-hunter.db` | SQLite database (auto-created, XDG-compliant) |

---

## External Services

### APIs & Integrations

| Service | Purpose | Documentation |
|---------|---------|---------------|
| Google Certificate Transparency Logs | Real-time CT log polling via RFC 6962 API | https://certificate.transparency.dev/ |
| Google Argon, Xenon, etc. (CT log shards) | Individual log endpoints polled by goroutines | https://www.gstatic.com/ct/log_list/v3/ |

### Third-Party Services

None. All data comes directly from public CT log infrastructure. No accounts, API keys, or authentication required.

---

## Security Considerations

### Authentication
- Not applicable — single-user local CLI tool, no network services exposed

### Data Protection
- Database is a local file with standard filesystem permissions
- No sensitive data stored — all data comes from public CT logs
- No credentials, API keys, or secrets needed to operate

### Dependencies
- `govulncheck` for vulnerability scanning of Go dependencies
- Minimal dependency surface — stdlib handles HTTP, x509, JSON
- All deps are well-established, actively maintained Go libraries

---

## Performance Targets

| Metric | Target | Measurement |
|--------|--------|-------------|
| Polling throughput | 300+ certs/sec across all logs | Certs processed per second counter in stats bar |
| TUI responsiveness | View switch < 50ms | No perceptible delay when toggling Live Feed ↔ DB Explorer |
| Query latency | < 1 second for typical filters | Time from command to output on 100k+ row database |
| Memory usage | < 100MB steady state | Runtime memory during sustained polling |
| Binary size | < 30MB | Compiled binary size (includes embedded SQLite) |

---

## Decision Log

| Date | Decision | Rationale | Alternatives Considered |
|------|----------|-----------|------------------------|
| 2026-02-24 | Go 1.26 | Latest stable, single binary distribution, goroutine concurrency model | Rust (steeper learning curve, slower iteration), Python (already prototyped, distribution pain) |
| 2026-02-24 | modernc.org/sqlite over mattn/go-sqlite3 | Pure Go, no CGo — enables simple cross-compilation | mattn/go-sqlite3 (requires C compiler), Bolt/bbolt (no SQL) |
| 2026-02-24 | Cobra for CLI | Standard Go subcommand framework, great UX out of the box | urfave/cli, stdlib flag |
| 2026-02-24 | Bubbletea + Lipgloss for TUI | Elm architecture decouples display from polling, most active Go TUI ecosystem | tview (less composable), tcell (too low-level) |
| 2026-02-24 | BurntSushi/toml for config | Battle-tested, minimal, TOML is human-friendly for keyword profiles | pelletier/go-toml, YAML (noisier syntax), JSON (not human-friendly for editing) |
| 2026-02-24 | stdlib log/slog for logging | Built-in structured logging since Go 1.21, zero deps | zerolog, zap (unnecessary deps for a CLI tool) |
| 2026-02-24 | testify for test assertions | Cleaner assertion syntax, widely used | pure stdlib (verbose), gocheck (less maintained) |

---

*Last updated: 2026-02-24*
