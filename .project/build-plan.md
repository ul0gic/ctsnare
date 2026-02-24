# ctsnare Build Plan

> **CRITICAL INSTRUCTIONS FOR ALL AGENTS**
>
> ## Project Structure
> All project documentation lives in the `.project/` directory at the repository root:
> ```
> .project/
> |-- prd.md           # Product Requirements Document
> |-- tech-stack.md    # Technology choices and rationale
> |-- build-plan.md    # This file - orchestration manifest and task tracking
> |-- changelog.md     # Version history and updates
> |-- issues/          # Issue tracking
> ```
>
> ## Build Discipline
> 1. **Keep this document up to date** -- Mark tasks as completed immediately after finishing them
> 2. **Build after every task** -- Run the build command after completing each task
> 3. **Zero tolerance for warnings/errors** -- Fix any warnings or errors before moving to the next task
> 4. **Update changelog.md** -- Log significant changes, fixes, and milestones
> 5. **Respect file ownership** -- Never modify files outside your assigned boundary during parallel phases
> 6. **Stop at merge gates** -- Do not proceed past a merge gate until the lead session completes the merge
>
> ```bash
> # Development (run directly)
> go run ./cmd/ctsnare
>
> # Build
> go build -o ctsnare ./cmd/ctsnare
>
> # Test
> go test ./...
>
> # Lint
> golangci-lint run ./...
>
> # Format
> gofmt -w .
>
> # Full verification (merge gates)
> go build -o ctsnare ./cmd/ctsnare && go vet ./... && golangci-lint run ./... && go test ./...
> ```
>
> If warnings or errors appear, fix them immediately. Do not proceed until the build is clean.
>
> ## Engineer Assignments
> Engineers are defined in `.claude/agents/`. Tasks are tagged with the responsible agent.
> Where marked PARALLEL, agents work simultaneously in isolated worktrees.
>
> | Icon | Agent | Color | Domain Boundary |
> |------|-------|-------|----------------|
> | BE | `backend-engineer` | Blue | `internal/poller/`, `internal/scoring/`, `internal/profile/`, `internal/storage/`, `internal/config/` |
> | CLI | `cli-engineer` | Green | `internal/tui/`, `internal/cmd/`, `cmd/ctsnare/` |
> | QA | `qa-engineer` | Green | `*_test.go` files across all packages (hardening phase only) |
> | SEC | `security-engineer` | Red | Security audit, hardening recommendations |
> | OPS | `devops-engineer` | Green | `.github/`, `Makefile`, `Dockerfile`, `.goreleaser.yml`, `.golangci.yml` (after foundation) |
> | DOC | `documentation-engineer` | Cyan | `README.md`, `docs/`, doc comments |
>
> ## Parallelization Strategy
> Phases marked with PARALLEL have independent workstreams executed by multiple agents concurrently.
> Each agent works in their own worktree during parallel phases. Work merges at defined merge gates.
> Tasks marked with [cloud] can be offloaded to cloud VMs. Tasks marked with [worktree] require worktree isolation.

---

## Orchestration Config

| Phase | Strategy | Worktrees | Agents | Cloud Eligible | Merge Gate After |
|-------|----------|-----------|--------|----------------|-----------------|
| Phase 1 | Sequential | No | Lead (BE) | No | -- |
| Phase 2 | PARALLEL | Yes | BE + CLI | No | Yes -- Merge Gate 1 |
| Phase 3 | Sequential | No | Lead (CLI) | No | -- |
| Phase 4 | PARALLEL | Yes | QA + SEC + OPS | Partial | Yes -- Merge Gate 2 |
| Phase 5 | Sequential | No | Lead (BE/CLI) | No | -- |
| Phase 6 | Sequential | No | DOC | No | -- |
| Phase 7.1 | Sequential | No | Lead (BE) | No | -- |
| Phase 7.2 | PARALLEL | Yes | BE + CLI | No | Yes -- Merge Gate 3 |
| Phase 7.3 | Sequential | No | Lead (CLI) | No | -- |
| Phase 7.4 | PARALLEL | Yes | QA + CLI | Partial | Yes -- Merge Gate 4 |

---

## Conflict Zones

> Files that more than one agent may need to touch. These are NEVER modified during parallel phases.
> All changes to conflict zone files happen at merge gates or in sequential phases.

| File/Path | Touched By | Resolution Strategy |
|-----------|-----------|-------------------|
| `go.mod` | BE, CLI, QA | Modify only in foundation phase (Phase 1). Collect new deps at merge gates, add in one commit |
| `go.sum` | BE, CLI, QA | Regenerate at merge gates via `go mod tidy` |
| `cmd/ctsnare/main.go` | BE, CLI | Scaffold in Phase 1. Only CLI agent modifies after that (owns Cobra root wiring) |
| `.golangci.yml` | BE, OPS | Create in Phase 1. Only OPS modifies after that |
| `.gitignore` | Lead only | Create in Phase 1. Modify only at merge gates |
| `.project/build-plan.md` | ALL agents | Every agent updates their own task statuses (`⬜` → `🔄` → `✅`). Append-only during parallel phases — only modify your own task rows |
| `.project/changelog.md` | ALL agents | Every agent logs milestones at end of sub-phases. Append-only |
| `.project/prd.md`, `.project/tech-stack.md` | Lead only | Only lead session modifies requirements and tech stack docs |
| `internal/cmd/root.go` | CLI (owner) | CLI agent owns all Cobra command files. BE never touches these |
| `internal/domain/types.go` | BE (Phase 7.1 only) | Modified in Phase 7.1 foundation to add enrichment and bookmark fields. FROZEN again before Phase 7.2 parallel work |
| `internal/domain/interfaces.go` | BE (Phase 7.1 only) | Modified in Phase 7.1 foundation to add enrichment-related store methods. FROZEN again before Phase 7.2 |
| `internal/domain/query.go` | BE (Phase 7.1 only) | Modified in Phase 7.1 foundation to add bookmark/liveness filter fields. FROZEN again before Phase 7.2 |
| `internal/storage/schema.go` | BE (Phase 7.1 only) | Modified in Phase 7.1 foundation to add enrichment and bookmark columns. FROZEN again before Phase 7.2 |
| `internal/tui/messages.go` | BE (Phase 7.1 only) | Modified in Phase 7.1 foundation to add new message types. FROZEN again before Phase 7.2 |

---

## Build Verification Protocol

> How and when builds are verified during parallel execution.

| Context | What to Run | Who Runs It |
|---------|------------|-------------|
| During Phase 2 -- BE agent | `go build ./internal/poller/... ./internal/scoring/... ./internal/profile/... ./internal/storage/... ./internal/config/... && go vet ./internal/poller/... ./internal/scoring/... ./internal/profile/... ./internal/storage/... ./internal/config/... && go test ./internal/poller/... ./internal/scoring/... ./internal/profile/... ./internal/storage/... ./internal/config/...` | BE in worktree |
| During Phase 2 -- CLI agent | `go build ./internal/tui/... ./internal/cmd/... ./cmd/ctsnare/... && go vet ./internal/tui/... ./internal/cmd/... ./cmd/ctsnare/... && go test ./internal/tui/... ./internal/cmd/...` | CLI in worktree |
| During Phase 4 -- QA agent | `go test -v -count=1 ./...` | QA in worktree |
| During Phase 4 -- OPS agent | `golangci-lint run ./... && go build -o ctsnare ./cmd/ctsnare` | OPS in worktree |
| During Phase 6 -- DOC agent | `make check` (full build + vet + lint + test after each doc comment batch). Also `go doc ./internal/<pkg>/` to verify doc comment rendering. | DOC on main |
| At merge gates | `go build -o ctsnare ./cmd/ctsnare && go vet ./... && golangci-lint run ./... && go test ./...` | Lead session on merged main |
| During Phase 7.2 -- BE agent | `go build ./internal/poller/... ./internal/scoring/... ./internal/profile/... ./internal/storage/... ./internal/config/... ./internal/enrichment/... && go vet ./internal/poller/... ./internal/scoring/... ./internal/profile/... ./internal/storage/... ./internal/config/... ./internal/enrichment/... && go test ./internal/poller/... ./internal/scoring/... ./internal/profile/... ./internal/storage/... ./internal/config/... ./internal/enrichment/...` | BE in worktree |
| During Phase 7.2 -- CLI agent | `go build ./internal/tui/... ./internal/cmd/... ./cmd/ctsnare/... && go vet ./internal/tui/... ./internal/cmd/... ./cmd/ctsnare/... && go test ./internal/tui/... ./internal/cmd/...` | CLI in worktree |
| During Phase 7.4 -- QA agent | `go test -v -count=1 -race ./...` | QA in worktree |
| During Phase 7.4 -- CLI agent | `go build ./internal/tui/... ./internal/cmd/... ./cmd/ctsnare/... && go vet ./internal/tui/... ./internal/cmd/... && go test ./internal/tui/... ./internal/cmd/...` | CLI in worktree |
| Before release | `go build -o ctsnare ./cmd/ctsnare && go vet ./... && golangci-lint run ./... && go test -race -count=1 ./...` | Lead session |

---

## Status Legend

| Icon | Status | Description |
|------|--------|-------------|
| ⬜ | Not Started | Task has not begun |
| 🔄 | In Progress | Currently being worked on |
| ✅ | Completed | Task finished |
| ⛔ | Blocked | Cannot proceed due to external dependency |
| ⚠️ | Has Blockers | Waiting on another task |
| 🔍 | In Review | Pending review/approval |
| 🚫 | Skipped | Intentionally not doing |
| ⏸️ | Deferred | Postponed to later phase/sprint |
| ☁️ | Cloud Eligible | Can be offloaded to cloud VM |
| 🌳 | Worktree | Requires worktree isolation |

---

## Project Progress Summary

```
Phase 1: Foundation          [████████████████████] 100%  ✅
Phase 2: Core Engine         [████████████████████] 100%  ✅
  🔀 Merge Gate 1            [████████████████████] 100%  ✅
Phase 3: Integration         [████████████████████] 100%  ✅
Phase 4: Hardening           [████████████████████] 100%  ✅
  🔀 Merge Gate 2            [████████████████████] 100%  ✅
Phase 5: Polish & Release    [████████████████████] 100%  ✅
Phase 6: Documentation       [████████████████████] 100%  ✅
Phase 7: Enhancements        [                    ]   0%
  7.1 Foundation             [                    ]   0%
  7.2 Parallel Build         [                    ]   0%
  🔀 Merge Gate 3            [                    ]   0%
  7.3 Integration            [                    ]   0%
  7.4 Parallel Polish        [                    ]   0%
  🔀 Merge Gate 4            [                    ]   0%
─────────────────────────────────────────────────────────────
Overall Progress             [███████████         ]  55%
```

| Phase | Tasks | Completed | Blocked | Deferred | Progress | Agents |
|-------|-------|-----------|---------|----------|----------|--------|
| Phase 1: Foundation | 15 | 15 | 0 | 0 | 100% | BE |
| Phase 2: Core Engine | 28 | 28 | 0 | 0 | 100% | BE, CLI |
| Merge Gate 1 | 1 | 1 | 0 | 0 | 100% | Lead |
| Phase 3: Integration | 10 | 10 | 0 | 0 | 100% | CLI |
| Phase 4: Hardening | 15 | 15 | 0 | 0 | 100% | QA, SEC, OPS |
| Merge Gate 2 | 1 | 1 | 0 | 0 | 100% | Lead |
| Phase 5: Polish & Release | 3 | 3 | 0 | 0 | 100% | BE, DOC |
| Phase 6: Documentation | 13 | 13 | 0 | 0 | 100% | DOC |
| Phase 7.1: Enhancement Foundation | 17 | 0 | 0 | 0 | 0% | BE |
| Phase 7.2: Enhancement Build | 32 | 0 | 0 | 0 | 0% | BE, CLI |
| Merge Gate 3 | 1 | 0 | 0 | 0 | 0% | Lead |
| Phase 7.3: Enhancement Integration | 8 | 0 | 0 | 0 | 0% | CLI |
| Phase 7.4: Enhancement Polish | 10 | 0 | 0 | 0 | 0% | QA, CLI |
| Merge Gate 4 | 1 | 0 | 0 | 0 | 0% | Lead |
| **Total** | **155** | **86** | **0** | **0** | **55%** | |

---

## Phase 1: Foundation

> Sequential. Lead session (backend-engineer) drives all initial setup. Must complete before any parallel work.
> No worktrees needed -- single working directory.
> **Risk: HIGH** -- Bad foundations poison everything downstream. Every decision here is load-bearing.

### File Ownership: Lead session (BE) owns everything during this phase.

### 1.1 Project Scaffold

| Status | Task | Description | Agent |
|--------|------|-------------|-------|
| ✅ | 1.1.1 | Initialize Go module: `go mod init github.com/ul0gic/ctsnare` in project root. Create `go.mod` with Go 1.26. | BE |
| ✅ | 1.1.2 | Create directory structure: `cmd/ctsnare/`, `internal/cmd/`, `internal/poller/`, `internal/scoring/`, `internal/profile/`, `internal/storage/`, `internal/tui/`, `internal/config/`. Each directory gets a placeholder `.go` file with the correct `package` declaration so the module compiles. | BE |
| ✅ | 1.1.3 | Create `cmd/ctsnare/main.go` -- minimal entry point that calls `internal/cmd.Execute()`. Just enough to compile: imports `internal/cmd`, calls `Execute()`, exits on error. | BE |
| ✅ | 1.1.4 | Create `.gitignore` with entries: `ctsnare` (binary), `*.db`, `*.sqlite`, `.env`, `dist/`, `coverage.out`, `*.prof`. | BE |
| ✅ | 1.1.5 | Create `.golangci.yml` with linters enabled: `errcheck`, `govet`, `staticcheck`, `unused`, `gosimple`, `ineffassign`, `gofmt`, `goimports`. Set `run.go` to `1.26`. Set `issues.max-issues-per-linter` to 0 (report all). | BE |
| ✅ | 1.1.6 | **BUILD CHECK** -- `go build ./cmd/ctsnare && go vet ./... && golangci-lint run ./...` passes clean with zero warnings. | BE |

### 1.2 Dependencies

| Status | Task | Description | Agent |
|--------|------|-------------|-------|
| ✅ | 1.2.1 | Install all production dependencies in one commit: `go get github.com/spf13/cobra@latest github.com/charmbracelet/bubbletea@latest github.com/charmbracelet/lipgloss@latest github.com/charmbracelet/bubbles@latest modernc.org/sqlite@latest github.com/BurntSushi/toml@latest`. Run `go mod tidy`. | BE |
| ✅ | 1.2.2 | Install dev dependencies: `go get github.com/stretchr/testify@latest`. Run `go mod tidy`. | BE |
| ✅ | 1.2.3 | **BUILD CHECK** -- `go build ./cmd/ctsnare && go mod verify` passes clean. Verify `go.sum` is populated. | BE |

### 1.3 Core Types & Interfaces

> These shared types are the contracts that all packages build against. They MUST be stable before parallel work begins.
> After this sub-phase, these types are FROZEN for Phase 2. Changes require a sequential phase.

| Status | Task | Description | Agent |
|--------|------|-------------|-------|
| ✅ | 1.3.1 | Create `internal/domain/types.go` (new package `domain`). Define core types: `Hit` struct (Domain, Score, Severity, Keywords []string, Issuer, IssuerCN, SANDomains []string, CertNotBefore time.Time, CTLog, Profile, Session, CreatedAt time.Time, UpdatedAt time.Time), `Severity` type (string const: HIGH, MED, LOW), `CTLogEntry` struct (raw entry from CT log before scoring: LeafInput []byte, ExtraData []byte, Index int64, LogURL string), `ScoredDomain` struct (Domain, Score int, Severity, MatchedKeywords []string -- output of scoring engine, input to storage). | BE |
| ✅ | 1.3.2 | Create `internal/domain/interfaces.go`. Define interfaces: `Scorer` (Score(domain string, profile *Profile) ScoredDomain), `Store` (InsertHit(ctx, Hit) error, QueryHits(ctx, QueryFilter) ([]Hit, error), UpsertHit(ctx, Hit) error, Stats(ctx) (DBStats, error), ClearAll(ctx) error, ClearSession(ctx, session string) error, Close() error), `ProfileLoader` (LoadProfile(name string) (*Profile, error), ListProfiles() []string). | BE |
| ✅ | 1.3.3 | Create `internal/domain/query.go`. Define `QueryFilter` struct (Keyword string, ScoreMin int, Severity string, Since time.Duration, TLD string, Session string, Limit int, Offset int, SortBy string, SortDir string) and `DBStats` struct (TotalHits int, BySeverity map[Severity]int, TopKeywords []KeywordCount, FirstHit time.Time, LastHit time.Time). Define `KeywordCount` struct (Keyword string, Count int). | BE |
| ✅ | 1.3.4 | Create `internal/domain/profile.go`. Define `Profile` struct (Name string, Keywords []string, SuspiciousTLDs []string, SkipSuffixes []string, Description string). This is the runtime representation of a keyword profile. | BE |
| ✅ | 1.3.5 | **BUILD CHECK** -- `go build ./... && go vet ./... && golangci-lint run ./...` passes clean. All domain types compile with no unused imports. | BE |

### 1.4 Cobra Root Command Scaffold

| Status | Task | Description | Agent |
|--------|------|-------------|-------|
| ✅ | 1.4.1 | Create `internal/cmd/root.go`. Define `rootCmd` with Cobra: `Use: "ctsnare"`, `Short: "Monitor Certificate Transparency logs for suspicious domains"`, `Long` with full description. Add persistent flags: `--config` (string, config file path), `--db` (string, database path override), `--verbose` (bool, enable debug logging). Create `Execute() error` function that runs rootCmd. Wire `cmd/ctsnare/main.go` to call this. | BE |
| ✅ | 1.4.2 | **BUILD CHECK** -- `go build -o ctsnare ./cmd/ctsnare && ./ctsnare --help` shows help output with the correct description and flags. `golangci-lint run ./...` passes clean. | BE |

---

## Phase 2: Core Engine -- PARALLEL

> PARALLEL. Two agents work simultaneously in isolated worktrees.
> Depends on: Phase 1 (all tasks complete, build passes clean)
>
> **BE agent** builds the data engine: CT log poller, scoring, profiles, SQLite storage, config.
> **CLI agent** builds the user interface: TUI dashboard (all views), Cobra subcommands, CLI output formatting.
>
> **Risk: MEDIUM** -- Agents work against frozen domain types from Phase 1. No shared file modifications.
> The CLI agent can stub storage calls using the `Store` interface from `internal/domain/interfaces.go`.
> The BE agent does NOT touch any file under `internal/tui/` or `internal/cmd/` (except it created root.go in Phase 1).
> The CLI agent does NOT touch any file under `internal/poller/`, `internal/scoring/`, `internal/profile/`, `internal/storage/`, `internal/config/`.

### File Ownership (This Phase)

| Agent | Owns (read/write) | Can Read (no write) |
|-------|-------------------|-------------------|
| BE (backend-engineer) | `internal/poller/`, `internal/scoring/`, `internal/profile/`, `internal/storage/`, `internal/config/` | `internal/domain/` (read only -- frozen) |
| CLI (cli-engineer) | `internal/tui/`, `internal/cmd/` (all subcommand files), `cmd/ctsnare/main.go` | `internal/domain/` (read only -- frozen) |

**OFF LIMITS during this phase:** `go.mod`, `go.sum`, `.golangci.yml`, `.gitignore`, `internal/domain/`, `.project/prd.md`, `.project/tech-stack.md`

### 2.1 Configuration System (BE) [worktree]

| Status | Task | Description | Agent |
|--------|------|-------------|-------|
| ✅ | 2.1.1 | Create `internal/config/config.go`. Define `Config` struct with all configurable values: `CTLogs []CTLogConfig` (URL string, Name string), `DefaultProfile string`, `BatchSize int` (default 256), `PollInterval time.Duration` (default 5s), `DBPath string` (default XDG-compliant `~/.local/share/ctsnare/ctsnare.db`), `CustomProfiles map[string]domain.Profile`, `SkipSuffixes []string`. Implement `Load(path string) (*Config, error)` that reads TOML file using `BurntSushi/toml`, applies defaults for missing values. Implement `DefaultConfig() *Config` with sensible defaults including 3 Google CT log URLs (Argon2025h1, Xenon2025h1, one more current shard). | BE |
| ✅ | 2.1.2 | Create `internal/config/config_test.go`. Test: loading valid TOML config, loading empty file uses defaults, loading non-existent file returns default config without error, invalid TOML returns error, CLI flag overrides (test the merge function). Use testify assertions. | BE |
| ✅ | 2.1.3 | **BUILD CHECK** -- `go test ./internal/config/... && go vet ./internal/config/...` passes clean. | BE |

### 2.2 Keyword Profiles (BE) [worktree]

| Status | Task | Description | Agent |
|--------|------|-------------|-------|
| ✅ | 2.2.1 | Create `internal/profile/builtin.go`. Define built-in profiles as Go constants/vars: `CryptoProfile` (keywords: casino, swap, exchange, airdrop, token, wallet, invest, mining, defi, stake, yield, claim, reward, bonus, crypto, bitcoin, ethereum, binance, coinbase, metamask; suspicious TLDs: .xyz, .top, .vip, .win, .bet, .casino, .click, .buzz, .icu, .monster), `PhishingProfile` (keywords: login, signin, verify, secure, account, update, confirm, banking, paypal, microsoft, apple, google, amazon, netflix, support, helpdesk, password, credential; suspicious TLDs: .xyz, .top, .info, .click, .buzz, .icu, .monster, .tk, .ml, .ga), `AllProfile` (combined keywords and TLDs from both). Each profile includes a `SkipSuffixes` list: cloudflaressl.com, amazonaws.com, herokuapp.com, azurewebsites.net, googleusercontent.com, fastly.net, akamaiedge.net, cloudfront.net, etc. | BE |
| ✅ | 2.2.2 | Create `internal/profile/profile.go`. Implement `Manager` struct that satisfies `domain.ProfileLoader` interface. `NewManager(customProfiles map[string]domain.Profile) *Manager`. Loads built-in profiles, merges in custom profiles from config. `LoadProfile(name string) (*domain.Profile, error)` returns the named profile or error. `ListProfiles() []string` returns sorted profile names. Custom profiles can extend built-in ones via an `Extends` field. | BE |
| ✅ | 2.2.3 | Create `internal/profile/profile_test.go`. Test: load each built-in profile by name, list returns all profiles sorted, load unknown profile returns error, custom profile extends built-in (inherits keywords + adds new ones), custom profile without extends starts fresh. | BE |
| ✅ | 2.2.4 | **BUILD CHECK** -- `go test ./internal/profile/... && go vet ./internal/profile/...` passes clean. | BE |

### 2.3 Scoring Engine (BE) [worktree]

| Status | Task | Description | Agent |
|--------|------|-------------|-------|
| ✅ | 2.3.1 | Create `internal/scoring/heuristics.go`. Implement individual scoring functions, each returns (score int, matched bool): `matchKeywords(domain string, keywords []string) (score int, matched []string)` -- 2 points per keyword found in domain (case-insensitive substring match), returns matched keywords. `scoreTLD(domain string, suspiciousTLDs []string) int` -- +1 if domain ends with a suspicious TLD. `scoreDomainLength(domain string) int` -- +1 if registered domain (excluding TLD) is over 30 chars. `scoreHyphenDensity(domain string) int` -- +1 if 2+ hyphens in the registered domain. `scoreNumberSequences(domain string) int` -- +1 if 4+ consecutive digits. `scoreMultiKeywordBonus(matchCount int) int` -- +2 if 3+ keywords matched. | BE |
| ✅ | 2.3.2 | Create `internal/scoring/scorer.go`. Implement `Engine` struct that satisfies `domain.Scorer` interface. `NewEngine() *Engine`. `Score(domain string, profile *domain.Profile) domain.ScoredDomain` runs all heuristics, sums score, classifies severity (HIGH >= 6, MED 4-5, LOW 1-3, skip if 0), checks skip suffixes first (return zero score if domain matches a skip suffix). Returns `ScoredDomain` with all fields populated. | BE |
| ✅ | 2.3.3 | Create `internal/scoring/scorer_test.go`. Table-driven tests covering: domain matching single keyword (score 2, LOW), domain matching 2 keywords (score 4, MED), domain matching 3+ keywords with bonus (score >= 6, HIGH), suspicious TLD adds +1, long domain adds +1, hyphen-heavy domain adds +1, number sequences adds +1, skip suffix domain returns zero score, case-insensitive matching works, empty profile returns zero, domain with all heuristics triggered gets maximum score. | BE |
| ✅ | 2.3.4 | **BUILD CHECK** -- `go test ./internal/scoring/... && go vet ./internal/scoring/...` passes clean. | BE |

### 2.4 SQLite Storage (BE) [worktree]

| Status | Task | Description | Agent |
|--------|------|-------------|-------|
| ✅ | 2.4.1 | Create `internal/storage/schema.go`. Define SQL schema as Go string constant: `hits` table with columns matching `domain.Hit` struct (id INTEGER PRIMARY KEY AUTOINCREMENT, domain TEXT NOT NULL UNIQUE, score INTEGER NOT NULL, severity TEXT NOT NULL, keywords TEXT NOT NULL -- JSON array, issuer TEXT, issuer_cn TEXT, san_domains TEXT -- JSON array, cert_not_before DATETIME, ct_log TEXT, profile TEXT, session TEXT DEFAULT '', created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP). Indexes: `idx_hits_score` (score DESC), `idx_hits_domain` (domain), `idx_hits_session` (session), `idx_hits_created_at` (created_at), `idx_hits_severity` (severity). | BE |
| ✅ | 2.4.2 | Create `internal/storage/db.go`. Implement `DB` struct wrapping `*sql.DB`. `NewDB(dbPath string) (*DB, error)` opens SQLite via `modernc.org/sqlite` driver, enables WAL mode (`PRAGMA journal_mode=WAL`), enables foreign keys, runs schema creation (CREATE TABLE IF NOT EXISTS). Implement `Close() error`. Handle XDG directory creation -- create parent directory of dbPath if it does not exist. | BE |
| ✅ | 2.4.3 | Create `internal/storage/hits.go`. Implement on `DB`: `UpsertHit(ctx context.Context, hit domain.Hit) error` -- INSERT OR REPLACE keyed on domain (deduplication), stores keywords and SANDomains as JSON arrays. `QueryHits(ctx context.Context, filter domain.QueryFilter) ([]domain.Hit, error)` -- builds SQL dynamically from filter fields (WHERE clauses for keyword LIKE, score >=, severity =, session =, created_at >= since, domain LIKE TLD pattern), applies ORDER BY (sortBy, sortDir), applies LIMIT/OFFSET. Reads JSON arrays back into Go slices. | BE |
| ✅ | 2.4.4 | Create `internal/storage/sessions.go`. Implement on `DB`: `ClearAll(ctx context.Context) error` -- DELETE FROM hits. `ClearSession(ctx context.Context, session string) error` -- DELETE FROM hits WHERE session = ?. `Stats(ctx context.Context) (domain.DBStats, error)` -- queries total count, count by severity, top 10 keywords (parse JSON arrays, count occurrences), first and last hit timestamps. | BE |
| ✅ | 2.4.5 | Create `internal/storage/export.go`. Implement on `DB`: `ExportJSONL(ctx context.Context, w io.Writer, filter domain.QueryFilter) error` -- writes one JSON line per hit to the writer. `ExportCSV(ctx context.Context, w io.Writer, filter domain.QueryFilter) error` -- writes CSV with header row. Both use QueryHits internally with no limit. | BE |
| ✅ | 2.4.6 | Create `internal/storage/db_test.go`. Tests using temporary database files (t.TempDir()): create database, insert hit, query it back, verify all fields roundtrip correctly. Test upsert: insert same domain twice, verify only one row exists with updated data. Test QueryHits with filters: keyword filter, score min filter, severity filter, session filter, sort order, limit/offset. Test ClearAll removes everything. Test ClearSession removes only matching session. Test Stats returns correct counts. Test export JSONL output format. | BE |
| ✅ | 2.4.7 | **BUILD CHECK** -- `go test ./internal/storage/... && go vet ./internal/storage/...` passes clean. | BE |

### 2.5 CT Log Poller (BE) [worktree]

| Status | Task | Description | Agent |
|--------|------|-------------|-------|
| ✅ | 2.5.1 | Create `internal/poller/ctlog.go`. Implement RFC 6962 API client: `CTLogClient` struct with `httpClient *http.Client` and `baseURL string`. Methods: `GetSTH(ctx context.Context) (*SignedTreeHead, error)` -- GET `<base>/ct/v1/get-sth`, parse JSON response (tree_size, timestamp, sha256_root_hash). `GetEntries(ctx context.Context, start, end int64) ([]domain.CTLogEntry, error)` -- GET `<base>/ct/v1/get-entries?start=N&end=N`, parse JSON entries array. Define `SignedTreeHead` struct (TreeSize int64, Timestamp int64). Handle HTTP errors, rate limiting (429 status with backoff), and non-200 responses with context-rich errors. | BE |
| ✅ | 2.5.2 | Create `internal/poller/parser.go`. Implement `ParseCertDomains(entry domain.CTLogEntry) ([]string, *x509.Certificate, error)`. Decodes the leaf_input (MerkleTreeLeaf -> TimestampedEntry -> ASN.1 certificate), parses x509 certificate using `crypto/x509`, extracts all domain names: Subject CommonName + all Subject Alternative Names (DNSNames). Deduplicates and returns unique domains. Handles parse errors gracefully (log warning, skip entry, don't crash). | BE |
| ✅ | 2.5.3 | Create `internal/poller/poller.go`. Implement `Poller` struct: `NewPoller(logURL string, logName string, scorer domain.Scorer, store domain.Store, profile *domain.Profile, batchSize int, pollInterval time.Duration, hitChan chan<- domain.Hit, statsChan chan<- PollStats) *Poller`. `Run(ctx context.Context) error` -- goroutine polling loop: get STH, fetch entries from last known position to tree_size in batches, parse certs, extract domains, score each domain, if score > 0 then upsert to store and send to hitChan, update position, sleep pollInterval, repeat. Respect context cancellation for graceful shutdown. Track stats: certs scanned, hits found, current position. Define `PollStats` struct (CertsScanned int64, HitsFound int64, CurrentIndex int64, TreeSize int64, LogName string). | BE |
| ✅ | 2.5.4 | Create `internal/poller/manager.go`. Implement `Manager` struct: `NewManager(cfg *config.Config, scorer domain.Scorer, store domain.Store, profile *domain.Profile) *Manager`. `Start(ctx context.Context, hitChan chan<- domain.Hit, statsChan chan<- PollStats) error` -- launches one Poller goroutine per CT log from config. Uses `errgroup.Group` for lifecycle management. Returns when context is cancelled. `Stop()` cancels context, waits for all pollers to exit. | BE |
| ✅ | 2.5.5 | Create `internal/poller/ctlog_test.go`. Test with httptest.NewServer mocking CT log responses: test GetSTH parses correctly, test GetEntries returns entries, test 429 rate limiting triggers backoff, test non-200 returns error, test invalid JSON returns error. | BE |
| ✅ | 2.5.6 | Create `internal/poller/parser_test.go`. Test certificate parsing with real-world certificate fixtures (base64-encoded leaf_input in testdata/). Test domain extraction includes CN and SANs. Test malformed certificate returns error without panic. Test certificate with no domains returns empty slice. | BE |
| ✅ | 2.5.7 | **BUILD CHECK** -- `go test ./internal/poller/... && go vet ./internal/poller/...` passes clean. | BE |

### 2.6 TUI Styles & Layout Foundation (CLI) [worktree]

| Status | Task | Description | Agent |
|--------|------|-------------|-------|
| ✅ | 2.6.1 | Create `internal/tui/styles.go`. Define all Lipgloss styles as package-level vars: `StyleHighSeverity` (bold red), `StyleMedSeverity` (bold yellow), `StyleLowSeverity` (green), `StyleHeader` (bold, border bottom), `StyleStatusBar` (dark background, light text, full width), `StyleSelectedRow` (inverted colors), `StyleHelpKey` (bold, muted foreground), `StyleHelpDesc` (muted foreground), `StyleTitle` (bold, padding), `StyleBorder` (rounded border). Use `lipgloss.AdaptiveColor` for light/dark terminal support. | CLI |
| ✅ | 2.6.2 | Create `internal/tui/keys.go`. Define key bindings using `bubbles/key` package: `KeyMap` struct with bindings for Quit (q), Tab (toggle views), Search (/), Sort (s), Filter (f), Enter (drill-down), Escape (back/dismiss), Clear (C), Help (?), Up/Down (k/j + arrows), PageUp/PageDown. Implement `ShortHelp()` and `FullHelp()` methods for the help bubble. | CLI |
| ✅ | 2.6.3 | **BUILD CHECK** -- `go build ./internal/tui/... && go vet ./internal/tui/...` passes clean. | CLI |

### 2.7 TUI Live Feed View (CLI) [worktree]

| Status | Task | Description | Agent |
|--------|------|-------------|-------|
| ✅ | 2.7.1 | Create `internal/tui/feed.go`. Implement `FeedModel` struct implementing `tea.Model`. State: `hits []domain.Hit` (circular buffer, most recent N hits for display), `viewport viewport.Model` (scrollable), `stats PollStats` (aggregate stats from all pollers), `topKeywords []domain.KeywordCount` (live-updating frequency). Renders: scrollable list of hits (each line: timestamp, severity tag color-coded, domain, score, matched keywords, issuer), severity tags use styles from 2.6.1. `Update` handles: `tea.WindowSizeMsg` (resize viewport), `HitMsg` (new hit arrived -- prepend to buffer, update top keywords), `StatsMsg` (update stats display), key messages for scrolling. `View` renders: header bar, hit list in viewport, stats bar at bottom (total certs scanned, hit count, certs/sec rate, active profile), top keywords sidebar on right if terminal width > 100. | CLI |
| ✅ | 2.7.2 | **BUILD CHECK** -- `go build ./internal/tui/... && go vet ./internal/tui/...` passes clean. | CLI |

### 2.8 TUI DB Explorer View (CLI) [worktree]

| Status | Task | Description | Agent |
|--------|------|-------------|-------|
| ✅ | 2.8.1 | Create `internal/tui/explorer.go`. Implement `ExplorerModel` struct implementing `tea.Model`. State: `table table.Model` (bubbles table), `hits []domain.Hit` (current result set from DB), `filter domain.QueryFilter` (current filter state), `sortCol int`, `sortDir string`, `loading bool`. Table columns: Severity, Score, Domain, Keywords, Issuer, Session, Timestamp. `Update` handles: `tea.WindowSizeMsg` (resize table), `tea.KeyMsg` (s to cycle sort column, / for search input, f for filter menu, Enter for drill-down, C for clear with confirmation), `HitsLoadedMsg` (populate table from DB query). Commands: `loadHitsCmd` queries the store with current filter, returns `HitsLoadedMsg`. Renders: filter status bar at top, table in center, help bar at bottom. | CLI |
| ✅ | 2.8.2 | Create `internal/tui/detail.go`. Implement `DetailModel` struct implementing `tea.Model`. Receives a single `domain.Hit` and renders full detail view: all SAN domains listed, complete issuer info (org + CN), certificate not-before timestamp, CT log name, profile that matched, session tag, all matched keywords, score breakdown. Uses Lipgloss bordered panel. Escape key returns to explorer. | CLI |
| ✅ | 2.8.3 | Create `internal/tui/filter.go`. Implement `FilterModel` for the filter input overlay: text inputs for keyword, min score, severity dropdown (HIGH/MED/LOW/all), time range (1h/6h/12h/24h/7d/all), session. Apply button commits filter and triggers a DB reload. Clear button resets all filters. | CLI |
| ✅ | 2.8.4 | **BUILD CHECK** -- `go build ./internal/tui/... && go vet ./internal/tui/...` passes clean. | CLI |

### 2.9 TUI Root App (CLI) [worktree]

| Status | Task | Description | Agent |
|--------|------|-------------|-------|
| ✅ | 2.9.1 | Create `internal/tui/app.go`. Implement `AppModel` struct (root tea.Model): State: `activeView int` (0=feed, 1=explorer, 2=detail), `feed FeedModel`, `explorer ExplorerModel`, `detail *DetailModel` (nil when not showing), `width int`, `height int`, `confirmingClear bool`. `Update` delegates to active view's Update, handles Tab (toggle activeView between feed and explorer), handles global quit (q). `View` renders active view. `Init` returns a batch of: subscribe to hit channel (tea.Cmd that reads from channel and sends HitMsg), subscribe to stats channel. Define custom messages: `HitMsg`, `StatsMsg`, `HitsLoadedMsg`, `SwitchViewMsg`, `ShowDetailMsg`. Create `NewApp(store domain.Store, hitChan <-chan domain.Hit, statsChan <-chan PollStats, profile string) AppModel`. | CLI |
| ✅ | 2.9.2 | Create `internal/tui/app_test.go`. Test model initialization, test view switching (Tab toggles between feed and explorer), test HitMsg updates feed model, test quit message. Use bubbletea testing utilities. | CLI |
| ✅ | 2.9.3 | **BUILD CHECK** -- `go build ./internal/tui/... && go test ./internal/tui/... && go vet ./internal/tui/...` passes clean. | CLI |

### 2.10 Cobra Subcommands (CLI) [worktree]

> The CLI agent builds all subcommands as Cobra commands. These reference domain types but use interface
> stubs for store/scorer where needed (actual wiring happens in Phase 3 integration).

| Status | Task | Description | Agent |
|--------|------|-------------|-------|
| ✅ | 2.10.1 | Create `internal/cmd/watch.go`. Define `watchCmd` cobra.Command: `Use: "watch"`, `Short: "Start live CT log monitoring"`. Flags: `--profile` (string, default "all"), `--session` (string, optional session tag), `--headless` (bool, no TUI -- poll and store silently), `--batch-size` (int, override config), `--poll-interval` (duration, override config). RunE function placeholder that creates config, creates store, creates scorer, creates profile manager, creates poller manager, and either starts TUI (with bubbletea) or runs headless (just pollers + graceful shutdown on SIGINT/SIGTERM). Add watchCmd to rootCmd. | CLI |
| ✅ | 2.10.2 | Create `internal/cmd/query.go`. Define `queryCmd` cobra.Command: `Use: "query"`, `Short: "Search and filter stored hits"`. Flags: `--keyword` (string), `--score-min` (int), `--since` (duration), `--tld` (string), `--session` (string), `--severity` (string), `--format` (string: table/json/csv, default table), `--limit` (int, default 50). RunE function: opens DB, builds QueryFilter from flags, calls store.QueryHits, formats output based on --format flag. Table format uses a simple tabwriter. JSON format outputs one JSON object per line. CSV format outputs with header. Add queryCmd to rootCmd. | CLI |
| ✅ | 2.10.3 | Create `internal/cmd/db.go`. Define `dbCmd` cobra.Command with subcommands: `db stats` (opens DB, calls store.Stats, prints formatted stats -- total hits, by severity, top keywords, date range), `db clear` (requires --confirm flag, optional --session flag, calls store.ClearAll or store.ClearSession), `db export` (--format jsonl/csv, --output file path or stdout, calls store.ExportJSONL/ExportCSV), `db path` (prints DB file path). Add dbCmd to rootCmd. | CLI |
| ✅ | 2.10.4 | Create `internal/cmd/profiles.go`. Define `profilesCmd` cobra.Command: `Use: "profiles"`, `Short: "List and inspect keyword profiles"`. No flags for list -- prints all profile names with descriptions. `profiles show <name>` flag: prints profile details (keywords, TLDs, skip suffixes). Add profilesCmd to rootCmd. | CLI |
| ✅ | 2.10.5 | Create `internal/cmd/output.go`. Implement shared output formatting: `FormatTable(hits []domain.Hit, w io.Writer) error` using tabwriter (columns: Severity, Score, Domain, Keywords, Issuer, Timestamp), `FormatJSON(hits []domain.Hit, w io.Writer) error` (one JSON object per line), `FormatCSV(hits []domain.Hit, w io.Writer) error` (CSV with header). Also `FormatStats(stats domain.DBStats, w io.Writer) error` for pretty-printing stats. | CLI |
| ✅ | 2.10.6 | **BUILD CHECK** -- `go build ./cmd/ctsnare && go vet ./internal/cmd/... && golangci-lint run ./internal/cmd/...` passes clean. `./ctsnare --help` shows all subcommands (watch, query, db, profiles). | CLI |

---

## Merge Gate 1: Post Phase 2

> All parallel work from Phase 2 STOPS here. No agent continues until merge is clean.

### Prerequisites
- ✅ BE (backend-engineer) has committed and pushed their worktree branch
- ✅ CLI (cli-engineer) has committed and pushed their worktree branch
- ✅ BE build verification passes in their worktree
- ✅ CLI build verification passes in their worktree

### Merge Protocol
1. Lead session creates fresh branch or works on main
2. Merge BE branch into main -- resolve any conflicts
3. Merge CLI branch into main -- resolve any conflicts
4. Run `go mod tidy` to reconcile any dependency differences
5. Run full verification: `go build -o ctsnare ./cmd/ctsnare && go vet ./... && golangci-lint run ./... && go test ./...`
6. Fix any issues before proceeding (failing agent fixes their code)
7. All agents pull fresh main before Phase 3

### Conflict Resolution Priority
1. `internal/domain/` -- canonical source of truth (should have no conflicts -- frozen in Phase 1)
2. `go.mod` / `go.sum` -- run `go mod tidy` on merged result
3. `cmd/ctsnare/main.go` -- CLI agent's version takes precedence (they own the entry point)
4. `internal/cmd/root.go` -- CLI agent's version takes precedence

### Post-Merge Verification
- ✅ `go build -o ctsnare ./cmd/ctsnare` succeeds
- ✅ `go vet ./...` reports zero issues
- ✅ `golangci-lint run ./...` reports zero issues
- ✅ `go test ./...` all tests pass
- ✅ `./ctsnare --help` shows all subcommands
- ✅ `./ctsnare watch --help` shows watch flags
- ✅ `./ctsnare query --help` shows query flags

---

## Phase 3: Integration

> Sequential. CLI agent (or lead) wires everything together.
> Depends on: Merge Gate 1 (must pass clean)
>
> This phase connects the components built in Phase 2: the watch command starts real pollers
> feeding real TUI, the query command reads from real storage, etc.
>
> **Risk: MEDIUM** -- This is where interface mismatches surface. Fix them here.

### File Ownership: CLI agent owns all files. BE agent available for consultation on storage/poller issues.

### 3.1 Wire Watch Command

| Status | Task | Description | Agent |
|--------|------|-------------|-------|
| ✅ | 3.1.1 | Update `internal/cmd/watch.go` RunE to wire real components: load config (internal/config), open storage (internal/storage), create scoring engine (internal/scoring), load profile (internal/profile), create hit and stats channels (buffered), create poller manager and start it, create TUI app model with store + channels + profile name, run `tea.NewProgram(app, tea.WithAltScreen())`. For headless mode: skip TUI, just run poller manager, block on context until SIGINT/SIGTERM. Handle graceful shutdown: context cancellation stops pollers, close DB on exit. | CLI |
| ✅ | 3.1.2 | Update `internal/cmd/query.go` RunE to wire real storage: load config for DB path, open storage, build QueryFilter from flags, call store.QueryHits, format output, close DB. Handle edge cases: no results found (print message, not error), DB file doesn't exist (friendly error message). | CLI |
| ✅ | 3.1.3 | Update `internal/cmd/db.go` subcommands to wire real storage: each subcommand opens DB via config, performs operation, closes DB. `db path` reads from config. `db export` writes to file or stdout based on --output flag. | CLI |
| ✅ | 3.1.4 | Update `internal/cmd/profiles.go` to wire real profile manager: create manager with config's custom profiles, list/show operations work against real data. | CLI |
| ✅ | 3.1.5 | **BUILD CHECK** -- `go build -o ctsnare ./cmd/ctsnare && go vet ./... && golangci-lint run ./... && go test ./...` passes clean. | CLI |

### 3.2 Graceful Shutdown & Signal Handling

| Status | Task | Description | Agent |
|--------|------|-------------|-------|
| ✅ | 3.2.1 | Implement signal handling in watch command: listen for SIGINT and SIGTERM, cancel context, wait for pollers to drain (with timeout), close database, restore terminal (bubbletea handles this for TUI mode). Log shutdown progress with slog. For headless mode, use `signal.NotifyContext`. | CLI |
| ✅ | 3.2.2 | Add structured logging throughout: use `log/slog` with JSON handler when --verbose is set, text handler otherwise. Log at appropriate levels: Info for startup/shutdown events, Debug for per-entry processing, Warn for skipped/malformed entries, Error for failures. Add slog initialization in root command PersistentPreRunE. | CLI |
| ✅ | 3.2.3 | **BUILD CHECK** -- `go build -o ctsnare ./cmd/ctsnare && go vet ./... && golangci-lint run ./... && go test ./...` passes clean. Manual smoke test: `./ctsnare watch --headless --verbose` starts without error, Ctrl-C shuts down cleanly. | CLI |

### 3.3 End-to-End Smoke Test

| Status | Task | Description | Agent |
|--------|------|-------------|-------|
| ✅ | 3.3.1 | Create `internal/cmd/integration_test.go`. Integration test: create temp DB, create config pointing to it, start watch in headless mode with a short timeout context (5 seconds), verify it starts and stops cleanly. Test query command with pre-populated DB (insert test hits, run query, verify output format). Test db stats with known data. Test profiles list output. | CLI |
| ✅ | 3.3.2 | **BUILD CHECK** -- `go build -o ctsnare ./cmd/ctsnare && go vet ./... && golangci-lint run ./... && go test -count=1 ./...` all pass. Binary runs and responds to --help, watch --help, query --help, db --help, profiles --help. | CLI |

---

## Phase 4: Hardening -- PARALLEL

> PARALLEL. Three agents work simultaneously on quality, security, and operations.
> Depends on: Phase 3 (all tasks complete, full build passes clean)
>
> **Risk: LOW** -- These agents add tests, audits, and CI/CD. They don't modify core logic.
> QA adds test files only. SEC produces audit report and may suggest code changes (filed as issues).
> OPS creates CI/CD, Makefile, Dockerfile -- all new files that don't conflict.

### File Ownership (This Phase)

| Agent | Owns (read/write) | Can Read (no write) |
|-------|-------------------|-------------------|
| QA (qa-engineer) | `*_test.go` files in all packages, `testdata/` directories | All source files (read only) |
| SEC (security-engineer) | `.project/issues/open/` (new issue files only) | All source files (read only) |
| OPS (devops-engineer) | `.github/workflows/`, `Makefile`, `Dockerfile`, `.goreleaser.yml` | All source files (read only) |

**OFF LIMITS during this phase:** `go.mod`, `go.sum`, all non-test `.go` files, `.golangci.yml`, `.project/prd.md`, `.project/tech-stack.md`

### 4.1 Test Coverage Expansion (QA) [worktree]

| Status | Task | Description | Agent |
|--------|------|-------------|-------|
| ✅ | 4.1.1 | Create `internal/scoring/heuristics_test.go`. Exhaustive table-driven tests for each heuristic function: keyword matching (case variations, partial matches, no match, empty input), TLD scoring (exact TLD match, subdomain of TLD, no match), domain length (exactly 30 chars, 31 chars, short domain), hyphen density (0, 1, 2, 3+ hyphens), number sequences (3 digits no score, 4 digits scores, 10 digits scores), multi-keyword bonus (0, 1, 2, 3, 5 keywords). | QA |
| ✅ | 4.1.2 | Create `internal/poller/manager_test.go`. Test manager starts and stops cleanly, test context cancellation stops all pollers, test stats aggregation from multiple pollers. Use mocked CT log server (httptest). | QA |
| ✅ | 4.1.3 | Create `internal/config/defaults_test.go`. Test that DefaultConfig returns valid config with all required fields populated: at least one CT log URL, valid batch size (>0), valid poll interval (>0), valid DB path, non-empty skip suffixes. Test XDG path construction. | QA |
| ✅ | 4.1.4 | Expand `internal/storage/db_test.go` with edge cases: concurrent read/write (goroutines inserting while querying), very long domain names, Unicode domains, empty keyword arrays, null-like fields, QueryFilter with all fields set simultaneously, pagination (limit + offset), sort by each column. | QA |
| ✅ | 4.1.5 | Create `internal/tui/feed_test.go`. Test FeedModel: hit buffer doesn't exceed max size, new hits prepend correctly, stats update correctly, viewport resize recalculates, severity styling applies correctly (unit test the View output for known hits). | QA |
| ✅ | 4.1.6 | **BUILD CHECK** -- `go test -v -count=1 -race ./...` all pass with zero failures. Run `go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out` and verify coverage is reported. | QA |

### 4.2 Security Audit (SEC) [worktree] [cloud]

| Status | Task | Description | Agent |
|--------|------|-------------|-------|
| ✅ | 4.2.1 | Audit `internal/storage/` for SQL injection: verify all queries use parameterized placeholders, no string concatenation for SQL, no fmt.Sprintf in query construction. Verify QueryHits dynamic query builder is safe. File issues for any findings. | SEC |
| ✅ | 4.2.2 | Audit `internal/poller/` for SSRF and input validation: verify CT log URLs are validated, verify certificate parsing handles malformed input without panic, verify HTTP client has timeouts set, verify no user-controlled URLs are fetched. File issues for any findings. | SEC |
| ✅ | 4.2.3 | Audit `internal/config/` for path traversal: verify DB path is sanitized, verify config file path is validated, verify XDG directory creation uses safe permissions (0700 for dirs, 0600 for files). Audit TOML parsing for denial of service (deeply nested, huge files). File issues for any findings. | SEC |
| ✅ | 4.2.4 | Run `govulncheck ./...` to check for known vulnerabilities in dependencies. Run `go mod verify` to check module integrity. File issues for any findings. | SEC |
| ✅ | 4.2.5 | **AUDIT REPORT** -- Write security assessment summary to `.project/issues/open/ISSUE-001-security-audit-v1.md` using the security report template. Include findings, risk ratings, and remediation recommendations. | SEC |

### 4.3 CI/CD & Build Infrastructure (OPS) [worktree] [cloud]

| Status | Task | Description | Agent |
|--------|------|-------------|-------|
| ✅ | 4.3.1 | Create `Makefile` with targets: `build` (go build -o ctsnare ./cmd/ctsnare), `test` (go test -race -count=1 ./...), `lint` (golangci-lint run ./...), `fmt` (gofmt -w .), `vet` (go vet ./...), `clean` (rm -f ctsnare *.db coverage.out), `coverage` (go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out), `check` (build + vet + lint + test -- full CI suite), `run` (go run ./cmd/ctsnare), `help` (print all targets). All targets use `.PHONY`. | OPS |
| ✅ | 4.3.2 | Create `.github/workflows/ci.yml`: trigger on push to main and pull_request. Jobs: `lint` (runs golangci-lint via golangci-lint-action), `test` (runs go test -race -count=1 ./... on ubuntu-latest with Go 1.26), `build` (runs go build, uploads binary as artifact). Cache Go modules. Matrix: test on ubuntu-latest. | OPS |
| ✅ | 4.3.3 | Create `.github/workflows/release.yml`: trigger on tag push (v*). Uses goreleaser to cross-compile for linux-amd64, linux-arm64, darwin-amd64, darwin-arm64, windows-amd64. Creates GitHub release with binaries and checksums. | OPS |
| ✅ | 4.3.4 | Create `.goreleaser.yml`: project name ctsnare, builds from ./cmd/ctsnare, GOOS/GOARCH matrix, archive format tar.gz (linux/mac) and zip (windows), checksum file, changelog from git log. | OPS |
| ✅ | 4.3.5 | **BUILD CHECK** -- `make check` passes clean (build + vet + lint + test). Verify CI workflow YAML is valid. | OPS |

---

## Merge Gate 2: Post Phase 4

> All parallel work from Phase 4 STOPS here. No agent continues until merge is clean.

### Prerequisites
- ✅ QA has committed and pushed their worktree branch (new test files only)
- ✅ SEC has committed audit report to `.project/issues/open/`
- ✅ OPS has committed and pushed their worktree branch (Makefile, CI/CD, goreleaser)
- ✅ QA build verification passes in their worktree
- ✅ OPS build verification passes in their worktree

### Merge Protocol
1. Lead session merges QA branch into main (test files only -- should be conflict-free)
2. Lead session merges SEC branch into main (issue files only -- should be conflict-free)
3. Lead session merges OPS branch into main (new files only -- should be conflict-free)
4. Run `go mod tidy` if any test dependencies were added
5. Run full verification: `make check` (or equivalent: build + vet + lint + test)
6. Address any critical security findings from SEC audit before proceeding
7. Fix any issues before proceeding
8. All agents pull fresh main before Phase 5

### Conflict Resolution Priority
1. Test files -- should never conflict (QA is the only agent writing tests in Phase 4)
2. `go.mod` / `go.sum` -- run `go mod tidy` on merged result
3. New files from OPS -- should never conflict (all new files)

### Post-Merge Verification
- ✅ `make check` passes (build + vet + lint + test with race detection)
- ✅ All new tests from QA pass (115+ tests, zero data races)
- ✅ `make build` produces working binary
- ✅ CI workflow files are present and valid YAML
- ✅ Security audit report is present in `.project/issues/open/` (1 HIGH, 2 MED, 3 LOW)

---

## Phase 5: Polish & Release Preparation

> Sequential. Security remediation and CLAUDE.md finalization.
> Depends on: Merge Gate 2 (must pass clean) + critical security findings addressed
>
> **Risk: LOW** -- Security fixes and internal documentation. Core functionality is complete and tested.

### File Ownership: Lead session (BE) owns all files.

### 5.1 Security Remediation

| Status | Task | Description | Agent |
|--------|------|-------------|-------|
| ✅ | 5.1.1 | Address all CRITICAL and HIGH findings from the Phase 4 security audit. Implement fixes as described in the issue files. Mark issues as resolved. Run full test suite after each fix. | BE |
| ✅ | 5.1.2 | **BUILD CHECK** -- `make check` passes clean after all security fixes. | BE |

### 5.2 Internal Documentation

| Status | Task | Description | Agent |
|--------|------|-------------|-------|
| ✅ | 5.2.1 | Update `/.claude/CLAUDE.md` with final project structure, build commands, and any new conventions established during development. Fill in all placeholder sections. | DOC |

---

## Phase 6: Documentation

> Sequential. Comprehensive documentation for public release.
> Depends on: Phase 5 (all tasks complete, build passes clean)
>
> This phase produces all user-facing and developer-facing documentation. The documentation-engineer
> reads the final codebase and produces accurate, current documentation based on what actually exists --
> not what was planned. Every code example must be verified against the built binary.
>
> **Risk: LOW** -- Documentation only. No source code modifications except adding Go doc comments to
> existing exported symbols. Build must remain green throughout.

### File Ownership: DOC (documentation-engineer) owns all files listed below.

| Agent | Owns (read/write) | Can Read (no write) |
|-------|-------------------|-------------------|
| DOC (documentation-engineer) | `README.md`, all `*.go` files (doc comments only), `.project/changelog.md` | All source files |

**OFF LIMITS during this phase:** `go.mod`, `go.sum`, `.golangci.yml`, `.goreleaser.yml`, `Makefile`, `.github/`, `.project/prd.md`, `.project/tech-stack.md`. DOC agent modifies `.go` files ONLY to add or improve doc comments on exported symbols -- no logic changes, no new functions, no refactoring.

### 6.1 README

| Status | Task | Description | Agent |
|--------|------|-------------|-------|
| ✅ | 6.1.1 | Create `README.md` at project root. Structure: (1) Project name + one-liner description ("Monitor Certificate Transparency logs for suspicious domains"), (2) Feature highlights (real-time CT polling, scoring engine, TUI dashboard, CLI query, SQLite storage, zero-config single binary), (3) Installation section with `go install github.com/ul0gic/ctsnare@latest` and binary download from GitHub Releases, (4) Quick Start section showing `ctsnare watch` and `ctsnare query` with expected terminal output. Keep this task focused on the top half of the README -- hero section through quick start. | DOC |
| ✅ | 6.1.2 | Continue `README.md` with full subcommand reference: `ctsnare watch` (all flags: --profile, --session, --headless, --batch-size, --poll-interval, with usage examples), `ctsnare query` (all flags: --keyword, --score-min, --since, --tld, --session, --severity, --format, --limit, with composable flag examples), `ctsnare db` (stats, clear, export, path subcommands with examples), `ctsnare profiles` (list, show subcommands with example output). Every command example must be copy-pasteable and show realistic output. | DOC |
| ✅ | 6.1.3 | Continue `README.md` with: (1) Configuration section -- TOML config file location (`~/.config/ctsnare/config.toml`), all configurable options with defaults and example config file, custom profile definition example, (2) Built-in Profiles section -- table of crypto, phishing, all profiles with keyword counts and descriptions, (3) Scoring section -- explain the scoring heuristics (keyword match, TLD, length, hyphens, digits, multi-keyword bonus) and severity thresholds (HIGH >= 6, MED 4-5, LOW 1-3). | DOC |
| ✅ | 6.1.4 | Continue `README.md` with: (1) Architecture section -- data flow diagram (ASCII art or Mermaid: CT Logs -> Pollers -> Scoring -> SQLite + TUI), key design decisions (decoupled polling/display, WAL mode, pure Go SQLite), (2) Development section -- prerequisites (Go 1.26, golangci-lint), clone + build + test commands, `make check` for full verification, project directory layout, (3) License section (MIT or as appropriate). | DOC |
| ✅ | 6.1.5 | **BUILD CHECK** -- `make check` passes clean. Verify `README.md` renders correctly: all code blocks have language tags, all links are valid, no broken markdown formatting. Run `./ctsnare --help` and verify README subcommand documentation matches actual CLI output. | DOC |

### 6.2 Go Doc Comments

> Go convention: every exported symbol gets a doc comment starting with the symbol name.
> Run `go doc ./internal/...` after each file to verify comments render correctly.
> Do NOT change any function signatures, logic, or behavior -- doc comments only.

| Status | Task | Description | Agent |
|--------|------|-------------|-------|
| ✅ | 6.2.1 | Add doc comments to all exported symbols in `internal/domain/`: `types.go` (Hit, Severity, SeverityHigh, SeverityMed, SeverityLow, CTLogEntry, ScoredDomain), `interfaces.go` (Scorer, Store, ProfileLoader -- document each interface method), `query.go` (QueryFilter -- document each field's purpose and expected values, DBStats, KeywordCount), `profile.go` (Profile -- document each field). Verify with `go doc ./internal/domain/`. | DOC |
| ✅ | 6.2.2 | Add doc comments to all exported symbols in `internal/config/config.go` (Config, CTLogConfig, Load, DefaultConfig, and any other exported functions/types), `internal/profile/profile.go` (Manager, NewManager, LoadProfile, ListProfiles), `internal/profile/builtin.go` (CryptoProfile, PhishingProfile, AllProfile, and any exported vars/consts). Verify with `go doc ./internal/config/ && go doc ./internal/profile/`. | DOC |
| ✅ | 6.2.3 | Add doc comments to all exported symbols in `internal/scoring/scorer.go` (Engine, NewEngine, Score method), `internal/scoring/heuristics.go` (any exported heuristic functions), `internal/storage/db.go` (DB, NewDB, Close), `internal/storage/hits.go` (UpsertHit, QueryHits), `internal/storage/sessions.go` (ClearAll, ClearSession, Stats), `internal/storage/export.go` (ExportJSONL, ExportCSV), `internal/storage/schema.go` (any exported schema constants). Verify with `go doc ./internal/scoring/ && go doc ./internal/storage/`. | DOC |
| ✅ | 6.2.4 | Add doc comments to all exported symbols in `internal/poller/ctlog.go` (CTLogClient, NewCTLogClient, GetSTH, GetEntries, SignedTreeHead), `internal/poller/parser.go` (ParseCertDomains), `internal/poller/poller.go` (Poller, NewPoller, Run, PollStats), `internal/poller/manager.go` (Manager, NewManager, Start, Stop). Add doc comments to all exported symbols in `internal/tui/` (AppModel, NewApp, FeedModel, ExplorerModel, DetailModel, FilterModel, all exported message types in messages.go, all exported style vars in styles.go, KeyMap in keys.go). Add doc comments to exported functions in `internal/cmd/` (Execute, and any exported output formatting functions in output.go). Verify with `go doc ./internal/poller/ && go doc ./internal/tui/ && go doc ./internal/cmd/`. | DOC |
| ✅ | 6.2.5 | **BUILD CHECK** -- `make check` passes clean after all doc comment additions. Run `go doc ./internal/domain/ && go doc ./internal/config/ && go doc ./internal/profile/ && go doc ./internal/scoring/ && go doc ./internal/storage/ && go doc ./internal/poller/ && go doc ./internal/tui/ && go doc ./internal/cmd/` and verify all exported symbols have doc comments with no warnings. | DOC |

### 6.3 Help Text & CLI Polish

| Status | Task | Description | Agent |
|--------|------|-------------|-------|
| ✅ | 6.3.1 | Review and improve all Cobra command help text. For each command (`root`, `watch`, `query`, `db`, `db stats`, `db clear`, `db export`, `db path`, `profiles`, `profiles list`, `profiles show`): verify `Short` is under 50 characters, verify `Long` includes a usage example, verify flag descriptions are clear and include default values where applicable. Run `./ctsnare --help`, `./ctsnare watch --help`, `./ctsnare query --help`, `./ctsnare db --help`, `./ctsnare profiles --help` and verify output is consistent and helpful. Fix any unclear or missing descriptions. | DOC |

### 6.4 Changelog & Release Milestone

| Status | Task | Description | Agent |
|--------|------|-------------|-------|
| ✅ | 6.4.1 | Update `.project/changelog.md`: consolidate all [Unreleased] sections into proper versioned entries. Add Phase 6 documentation work. Mark milestone `0.5.0` as Feature Complete with today's date in the Milestones table. Ensure the changelog follows Keep a Changelog format consistently -- every entry categorized (Added, Changed, Fixed, Security), reverse chronological order, version headers link to git tags. | DOC |
| ✅ | 6.4.2 | **BUILD CHECK** -- Final full verification: `make check` passes clean. Run `./ctsnare --help` and verify all commands are documented. Run `go doc ./...` and verify all exported symbols have doc comments. Verify `README.md` exists and is well-formed. Verify `.project/changelog.md` has 0.5.0 milestone entry. This is the final quality gate before the project is considered feature complete. | DOC |

---

## Phase 7: Enhancements

> Multi-phase enhancement cycle that adds enrichment, TUI polish, backtracking, batch operations, and bookmarking.
> Depends on: Phase 6 (all tasks complete, build passes clean)
>
> This phase unfreezes `internal/domain/` types for the first time since Phase 1. Domain type changes are
> made in Phase 7.1 (sequential foundation) and then REFROZEN before Phase 7.2 parallel work begins.
> Phase 7 follows the same foundation-then-parallel pattern as the original build but scoped to enhancements.
>
> **Risk: MEDIUM** -- Domain type modifications ripple across storage, poller, TUI, and CLI.
> The foundation sub-phase (7.1) must be thorough and the schema migration must be backward-compatible.

---

### Phase 7.1: Enhancement Foundation (Sequential)

> Sequential. Lead session (backend-engineer) drives all schema, type, and interface changes.
> No worktrees needed -- single working directory on main.
> Must complete before any Phase 7.2 parallel work.
>
> **CRITICAL: Domain types are UNFROZEN for this sub-phase only.** After 7.1 completes, domain types
> are REFROZEN for all of Phase 7.2 and beyond. This is the only window to modify shared contracts.
>
> **Risk: HIGH** -- These changes affect every package that imports `internal/domain/`. Every addition
> must be backward-compatible (new fields with zero-value defaults, new optional interface methods).

#### File Ownership: Lead session (BE) owns everything during this sub-phase.

#### 7.1.1 Domain Type Extensions

| Status | Task | Description | Agent |
|--------|------|-------------|-------|
| ✅ | 7.1.1.1 | Extend `internal/domain/types.go`: Add enrichment fields to the `Hit` struct: `IsLive bool` (domain responded to HTTP probe), `ResolvedIPs []string` (DNS A/AAAA records), `HostingProvider string` (detected CDN/host from reverse DNS or IP range), `HTTPStatus int` (status code from liveness probe), `LiveCheckedAt time.Time` (when the probe ran), `Bookmarked bool` (user-flagged as interesting). All new fields have zero-value defaults so existing code continues to work without modification. | BE |
| ✅ | 7.1.1.2 | Extend `internal/domain/query.go`: Add new filter fields to `QueryFilter` struct: `Bookmarked bool` (filter to bookmarked-only hits), `LiveOnly bool` (filter to live domains only). These fields default to false (no filter). | BE |
| ✅ | 7.1.1.3 | Extend `internal/domain/interfaces.go`: Add new methods to the `Store` interface: `SetBookmark(ctx context.Context, domain string, bookmarked bool) error`, `DeleteHit(ctx context.Context, domain string) error`, `DeleteHits(ctx context.Context, domains []string) error`, `UpdateEnrichment(ctx context.Context, domain string, isLive bool, resolvedIPs []string, hostingProvider string, httpStatus int) error`. These are additive methods -- existing interface implementations must add them. | BE |
| ✅ | 7.1.1.4 | **BUILD CHECK** -- `go build ./... && go vet ./... && golangci-lint run ./...` passes clean. Existing tests may fail at this point due to interface expansion -- that is expected and will be fixed in 7.1.2. | BE |

#### 7.1.2 Storage Schema Migration

| Status | Task | Description | Agent |
|--------|------|-------------|-------|
| ✅ | 7.1.2.1 | Update `internal/storage/schema.go`: Add new columns to the `hits` table schema using `ALTER TABLE IF NOT EXISTS` pattern or a migration approach. New columns: `is_live INTEGER DEFAULT 0`, `resolved_ips TEXT DEFAULT '[]'` (JSON array), `hosting_provider TEXT DEFAULT ''`, `http_status INTEGER DEFAULT 0`, `live_checked_at DATETIME`, `bookmarked INTEGER DEFAULT 0`. Add indexes: `idx_hits_bookmarked` on `(bookmarked)` where bookmarked=1, `idx_hits_is_live` on `(is_live)`. Use a separate migration SQL constant (e.g., `migrationV2SQL`) that runs ALTER TABLE statements wrapped in try/ignore-if-exists logic so the migration is idempotent. Call this migration from `NewDB` after the initial schema creation. | BE |
| ✅ | 7.1.2.2 | Update `internal/storage/hits.go`: Extend `UpsertHit` to write the new enrichment and bookmark columns. Extend `QueryHits` to support the new filter fields (`Bookmarked`, `LiveOnly`). Extend `scanHit` to read the new columns back into the extended `Hit` struct (handle `resolved_ips` as JSON array, `live_checked_at` as timestamp). Extend `sanitizeSortColumn` to accept `is_live`, `bookmarked`, `http_status`, `live_checked_at` as valid sort columns. | BE |
| ✅ | 7.1.2.3 | Implement `SetBookmark` on `*DB`: `UPDATE hits SET bookmarked = ? WHERE domain = ?`. Implement `DeleteHit` on `*DB`: `DELETE FROM hits WHERE domain = ?`. Implement `DeleteHits` on `*DB`: `DELETE FROM hits WHERE domain IN (?)` -- use a transaction with batched parameter binding for the domain list. Implement `UpdateEnrichment` on `*DB`: `UPDATE hits SET is_live = ?, resolved_ips = ?, hosting_provider = ?, http_status = ?, live_checked_at = ? WHERE domain = ?` -- serialize `resolved_ips` as JSON array. All methods in `internal/storage/hits.go`. | BE |
| ✅ | 7.1.2.4 | Update `internal/storage/db_test.go`: Add tests for the new schema migration (open existing DB, verify new columns exist). Test `SetBookmark` (bookmark a hit, query with bookmarked filter, verify). Test `DeleteHit` (insert hit, delete it, verify gone). Test `DeleteHits` (insert 5 hits, delete 3, verify 2 remain). Test `UpdateEnrichment` (insert hit, update enrichment fields, query back, verify all fields roundtrip including resolved_ips JSON and live_checked_at timestamp). Test that `QueryHits` with `Bookmarked: true` returns only bookmarked hits. Test that `QueryHits` with `LiveOnly: true` returns only live hits. | BE |
| ✅ | 7.1.2.5 | **BUILD CHECK** -- `go build ./... && go vet ./... && golangci-lint run ./... && go test ./...` passes clean. All existing tests plus new migration tests pass. | BE |

#### 7.1.3 TUI Message Types & Shared Contracts

| Status | Task | Description | Agent |
|--------|------|-------------|-------|
| ✅ | 7.1.3.1 | Update `internal/tui/messages.go`: Add new message types: `EnrichmentMsg struct { Domain string; IsLive bool; ResolvedIPs []string; HostingProvider string; HTTPStatus int }` (enrichment result arrived for a domain), `BookmarkToggleMsg struct { Domain string; Bookmarked bool }` (bookmark state changed), `DeleteHitsMsg struct { Domains []string }` (hits were deleted), `DiscardedDomainMsg struct { Domain string }` (domain was scanned but scored zero -- for the activity feed). Add `HitsPerMin float64` field to the existing `PollStats` struct. | BE |
| ✅ | 7.1.3.2 | Update `internal/tui/styles.go`: Add new style constants: `StyleLiveDomain` (bold green foreground, for highlighting live domains), `StyleDiscardedDomain` (dim gray foreground, for briefly showing discarded domains in the feed), `StyleBookmarked` (gold/yellow star icon style), `StyleSelectedCheckbox` (for multi-select visual indicator in explorer), `colorLive` adaptive color (green), `colorDiscarded` adaptive color (dark gray/dim). | BE |
| ✅ | 7.1.3.3 | Update `internal/tui/keys.go`: Add new key bindings to `KeyMap`: `Bookmark` (key: "b", help: "bookmark"), `Delete` (key: "d", help: "delete"), `SelectToggle` (key: " " (space), help: "select"), `SelectAll` (key: "a", help: "select all"), `DeselectAll` (key: "A", help: "deselect all"), `ConfirmDelete` (key: "D", help: "delete selected"). Update `ShortHelp()` and `FullHelp()` to include the new bindings. | BE |
| ✅ | 7.1.3.4 | **BUILD CHECK** -- `go build ./... && go vet ./... && golangci-lint run ./... && go test ./...` passes clean. All message types, styles, and key bindings compile without errors. | BE |

#### 7.1.4 Enrichment Package Scaffold

| Status | Task | Description | Agent |
|--------|------|-------------|-------|
| ✅ | 7.1.4.1 | Create directory `internal/enrichment/` and file `internal/enrichment/enricher.go`. Define the `Enricher` struct: holds a `domain.Store`, `httpClient *http.Client` (with 5s timeout), rate limiter state (max 5 concurrent probes, 1 req/sec per domain via `golang.org/x/time/rate` or a simple semaphore+ticker). Define `NewEnricher(store domain.Store, enrichChan chan<- EnrichResult) *Enricher`. Define `EnrichResult` struct: `Domain string`, `IsLive bool`, `ResolvedIPs []string`, `HostingProvider string`, `HTTPStatus int`, `Error error`. Define `Enqueue(domain string)` method that adds a domain to the probe queue. Define `Run(ctx context.Context) error` method signature (goroutine that drains the queue, probes domains, writes results to store, sends to enrichChan). Implement the rate-limited worker pool: a buffered queue channel (capacity 1000), 5 worker goroutines that each rate-limit to 1 req/sec. Placeholder implementations for DNS and HTTP probe (to be filled in 7.2). | BE |
| ✅ | 7.1.4.2 | Create `internal/enrichment/dns.go`. Define `ResolveDomain(domain string) (ips []string, provider string, err error)`. Uses `net.DefaultResolver.LookupHost` to resolve A/AAAA records. Detects hosting provider by checking resolved IPs against known CIDR ranges or reverse DNS patterns: Cloudflare (104.16.0.0/12, 172.64.0.0/13, or reverse DNS containing "cloudflare"), AWS (*.amazonaws.com reverse DNS), Google Cloud (*.googleusercontent.com, *.1e100.net), Azure (*.azure.com), Fastly, Akamai, DigitalOcean (*.digitalocean.com). Falls back to reverse DNS lookup (`net.LookupAddr`) if no CIDR match. Returns "unknown" if no provider detected. Short timeout (3s) per resolution to avoid blocking the enrichment pipeline. | BE |
| ✅ | 7.1.4.3 | Create `internal/enrichment/http.go`. Define `ProbeLiveness(domain string) (statusCode int, isLive bool, err error)`. Sends HTTP HEAD request to `https://<domain>/` with a 5-second timeout. If HTTPS fails (connection refused, TLS error), try `http://<domain>/` as fallback. `isLive` is true if any HTTP response is received (even 4xx/5xx -- the domain resolves and has a web server). Returns status code 0 and isLive=false if both attempts fail or timeout. Set `User-Agent` header to a reasonable value (e.g., `ctsnare/1.0 (domain-liveness-check)`). Follow up to 3 redirects. Do NOT read the response body (HEAD request only). | BE |
| ✅ | 7.1.4.4 | **BUILD CHECK** -- `go build ./... && go vet ./... && golangci-lint run ./...` passes clean. The enrichment package compiles. Tests will be added during Phase 7.2. | BE |

---

### Phase 7.2: Enhancement Build -- PARALLEL

> PARALLEL. Two agents work simultaneously in isolated worktrees.
> Depends on: Phase 7.1 (all tasks complete, build passes clean)
>
> **BE agent** builds the enrichment pipeline, backtrack mode, and storage integration.
> **CLI agent** builds the TUI visual overhaul, batch delete, bookmark UI, and enrichment display.
>
> **Risk: MEDIUM** -- Agents work against the extended domain types from Phase 7.1. No shared file modifications.
> The CLI agent reads from `internal/domain/` and `internal/enrichment/` (read only -- frozen).
> The BE agent does NOT touch any file under `internal/tui/` or `internal/cmd/`.
> The CLI agent does NOT touch any file under `internal/poller/`, `internal/scoring/`, `internal/storage/`, `internal/config/`, `internal/enrichment/`.

#### File Ownership (This Phase)

| Agent | Owns (read/write) | Can Read (no write) |
|-------|-------------------|-------------------|
| BE (backend-engineer) | `internal/poller/`, `internal/scoring/`, `internal/storage/`, `internal/config/`, `internal/enrichment/` | `internal/domain/` (read only -- frozen), `internal/tui/messages.go` (read only) |
| CLI (cli-engineer) | `internal/tui/`, `internal/cmd/`, `cmd/ctsnare/` | `internal/domain/` (read only -- frozen), `internal/enrichment/` (read only -- types only) |

**OFF LIMITS during this phase:** `go.mod`, `go.sum`, `.golangci.yml`, `.gitignore`, `internal/domain/`, `internal/tui/messages.go`, `internal/tui/styles.go`, `internal/tui/keys.go`, `.project/prd.md`, `.project/tech-stack.md`

#### 7.2.1 Enrichment Pipeline (BE) [worktree]

| Status | Task | Description | Agent |
|--------|------|-------------|-------|
| ✅ | 7.2.1.1 | Implement the full enrichment worker pool in `internal/enrichment/enricher.go`. The `Run` method starts 5 worker goroutines. Each worker reads domains from a buffered channel (capacity 1000), calls `ResolveDomain` and `ProbeLiveness`, writes results to the store via `store.UpdateEnrichment`, and sends `EnrichResult` to the enrichChan. Rate limiting: use a `time.Ticker` with 1-second interval per worker (so aggregate rate is 5 req/sec). Handle context cancellation for graceful shutdown. `Enqueue` is non-blocking -- if the queue is full, log a warning and drop (the domain is already stored, enrichment is best-effort). | BE |
| ✅ | 7.2.1.2 | Create `internal/enrichment/enricher_test.go`. Test with httptest.NewServer: test that a live domain returns `isLive=true` and a valid HTTP status. Test that a non-existent domain returns `isLive=false`. Test DNS resolution returns IP addresses for a known domain (use localhost/127.0.0.1 for controlled tests). Test rate limiting: enqueue 20 domains, verify they complete but don't all fire simultaneously (measure elapsed time > 3s). Test graceful shutdown: cancel context, verify workers exit without panic. Test queue overflow: enqueue more than capacity, verify no blocking or panic. | BE |
| ✅ | 7.2.1.3 | Create `internal/enrichment/dns_test.go`. Test `ResolveDomain` for localhost (should return 127.0.0.1 and provider "unknown"). Test provider detection logic with known IP ranges (mock or use table-driven tests with specific IPs that fall into Cloudflare/AWS/Google ranges). Test that timeout is respected (mock a slow DNS server or use a very short timeout). Test reverse DNS fallback. | BE |
| ✅ | 7.2.1.4 | Create `internal/enrichment/http_test.go`. Test `ProbeLiveness` against httptest.NewServer (returns isLive=true, correct status code). Test against a server that only responds on HTTP (HTTPS fallback to HTTP works). Test against no server (returns isLive=false, status 0). Test redirect following (up to 3 redirects). Test timeout behavior (slow server, verify timeout fires). | BE |
| ✅ | 7.2.1.5 | **BUILD CHECK** -- `go test ./internal/enrichment/... && go vet ./internal/enrichment/...` passes clean. | BE |

#### 7.2.2 Backtrack Mode (BE) [worktree]

| Status | Task | Description | Agent |
|--------|------|-------------|-------|
| ✅ | 7.2.2.1 | Update `internal/poller/poller.go`: Add a `backtrack int64` field to the `Poller` struct. Update `NewPoller` to accept a `backtrack int64` parameter. In the `Run` method, after getting the initial STH, set `currentIndex = sth.TreeSize - backtrack` (clamped to 0 minimum). When `backtrack > 0`, the poller starts behind the tip and works forward, giving immediate results on launch. When `backtrack == 0` (default), behavior is unchanged -- start at the tip, wait for new entries. | BE |
| ✅ | 7.2.2.2 | Update `internal/poller/manager.go`: Add a `Backtrack int64` field to the `Manager` struct. Pass it through to each `NewPoller` call in `Start`. Update `NewManager` to accept the backtrack value from config or CLI flag. | BE |
| ✅ | 7.2.2.3 | Update `internal/config/config.go`: Add `Backtrack int64` field to `Config` struct with TOML tag `backtrack`. Default value: 0. Update `MergeFlags` to accept and apply a backtrack override. | BE |
| ✅ | 7.2.2.4 | Create tests in `internal/poller/poller_test.go` (or extend existing): Test that when backtrack=1000 and tree_size=5000, the poller starts at index 4000. Test that when backtrack=0, the poller starts at tree_size (current behavior). Test that when backtrack > tree_size, the poller starts at index 0 (clamped). Use httptest mock CT log server. | BE |
| ✅ | 7.2.2.5 | **BUILD CHECK** -- `go test ./internal/poller/... ./internal/config/... && go vet ./internal/poller/... ./internal/config/...` passes clean. | BE |

#### 7.2.3 Storage & Export Extensions (BE) [worktree]

| Status | Task | Description | Agent |
|--------|------|-------------|-------|
| ✅ | 7.2.3.1 | Update `internal/storage/export.go`: Extend `ExportJSONL` and `ExportCSV` to include the new enrichment fields (is_live, resolved_ips, hosting_provider, http_status, live_checked_at) and the bookmarked field in the output. CSV gets new columns at the end (backward-compatible for parsers that use column names). JSONL gets the new fields in each JSON object. | BE |
| ✅ | 7.2.3.2 | Update `internal/storage/db_test.go`: Add roundtrip tests for the enrichment fields through export: insert hit with enrichment data, export as JSONL, parse and verify all fields present. Same for CSV. Verify backward compatibility -- a hit with zero-value enrichment fields exports cleanly (empty arrays, zero values). | BE |
| ✅ | 7.2.3.3 | **BUILD CHECK** -- `go test ./internal/storage/... && go vet ./internal/storage/...` passes clean. | BE |

#### 7.2.4 TUI Visual Overhaul (CLI) [worktree]

| Status | Task | Description | Agent |
|--------|------|-------------|-------|
| ⬜ | 7.2.4.1 | Overhaul `internal/tui/feed.go` -- severity colors throughout: Update `renderHitLine` to apply full-line severity coloring -- RED foreground for HIGH hits, ORANGE for MED, YELLOW/GREEN for LOW. The severity tag `[HIGH]`, `[MED]`, `[LOW]` should be rendered with the existing severity styles (which already map to red/yellow/green) but now also tint the entire domain text to match the severity color. Score number should also be colored to match severity. | CLI |
| ⬜ | 7.2.4.2 | Overhaul `internal/tui/feed.go` -- auto-scrolling activity feed: Add a `discards []discardEntry` field (struct: Domain string, FadeAt time.Time) that holds recently discarded domains (scored zero). When a `DiscardedDomainMsg` arrives, prepend to the discards buffer (max 50 entries, display for 2 seconds then fade). In `renderHits`, interleave discarded domains (rendered in `StyleDiscardedDomain` -- dim gray) between real hits to show constant activity. Add a tick command that fires every 500ms to clear expired discards and refresh the viewport. This makes the feed feel alive even when hit rate is low. | CLI |
| ⬜ | 7.2.4.3 | Overhaul `internal/tui/feed.go` -- enhanced status bar and throughput stats: Update `renderStatusBar` to show: `Scanned: N | Hits: N | Rate: N.N certs/s | Hits/min: N.N | Discarded: N | Logs: N | Profile: name`. Add `discardCount int64` to FeedModel to track total discards. Calculate hits/min from `PollStats.HitsPerMin`. Use color coding in the status bar: green for rate numbers, yellow for hit counts, red when rate drops to zero. | CLI |
| ⬜ | 7.2.4.4 | Add help bar to `internal/tui/feed.go`: Below the status bar (or integrated into it), render a single-line help bar showing active keybindings: `Tab=views | q=quit | ?=help | j/k=scroll`. Use `StyleHelpKey` and `StyleHelpDesc` for consistent styling. The help bar should be aware of the current view context. | CLI |
| ⬜ | 7.2.4.5 | Update `internal/tui/explorer.go` -- severity colors in table: Override the table row rendering to apply severity colors to the Severity column cells. HIGH rows show the severity cell in red, MED in orange/yellow, LOW in green. The Score column should also be tinted to match. Use the table's `StyleFunc` option (if available in bubbles table) or post-process the rendered output with ANSI color codes per severity. | CLI |
| ⬜ | 7.2.4.6 | Update `internal/tui/detail.go` -- severity colors and enrichment display: The severity field should render in its color (already partially done). Add a new section "Enrichment Data" below the existing fields: show `Live: Yes/No` (green/red colored), `Resolved IPs: 1.2.3.4, 5.6.7.8`, `Hosting Provider: Cloudflare`, `HTTP Status: 200`, `Last Checked: 2026-02-24 14:30:00`. Show `Bookmarked: *` (gold star if bookmarked). Only display enrichment section if `LiveCheckedAt` is non-zero (enrichment has run). | CLI |
| ⬜ | 7.2.4.7 | **BUILD CHECK** -- `go build ./internal/tui/... && go vet ./internal/tui/...` passes clean. | CLI |

#### 7.2.5 Batch Delete & Selection (CLI) [worktree]

| Status | Task | Description | Agent |
|--------|------|-------------|-------|
| ⬜ | 7.2.5.1 | Update `internal/tui/explorer.go` -- multi-select state: Add `selected map[int]bool` field to `ExplorerModel` (maps row index to selected state). Add `selectAllActive bool` field. Handle spacebar key: toggle current row in `selected` map, move cursor down one row. Handle `a` key: select all visible rows (set every index in `selected`). Handle `A` key: deselect all (clear `selected` map). Update `hitsToRows` to prepend a checkbox column: `[x]` for selected rows, `[ ]` for unselected. Add the checkbox column to the table column definitions (width 4, first column). | CLI |
| ⬜ | 7.2.5.2 | Update `internal/tui/explorer.go` -- delete operations: Handle `d` key: if current row is highlighted, show confirmation prompt "Delete hit for <domain>? (y/n)". On `y`, send a `tea.Cmd` that calls `store.DeleteHit(ctx, domain)` and then reloads hits. Handle `D` key: if `selected` has entries, show confirmation prompt "Delete N selected hits? (y/n)". On `y`, collect domains from selected indices, send a `tea.Cmd` that calls `store.DeleteHits(ctx, domains)`, clear `selected` map, reload hits. Add a `confirmAction string` field to ExplorerModel for the confirmation overlay state. Render the confirmation prompt as a simple overlay at the bottom of the table. | CLI |
| ⬜ | 7.2.5.3 | Update `internal/tui/explorer.go` -- clear all with confirmation: Update the existing `C` key handler to show a confirmation prompt "Clear ALL hits? This cannot be undone. (y/n)". On `y`, call `store.ClearAll(ctx)` and reload. This replaces any existing clear behavior with a properly confirmed version. | CLI |
| ⬜ | 7.2.5.4 | **BUILD CHECK** -- `go build ./internal/tui/... && go vet ./internal/tui/...` passes clean. | CLI |

#### 7.2.6 Bookmark/Flag System -- TUI (CLI) [worktree]

| Status | Task | Description | Agent |
|--------|------|-------------|-------|
| ⬜ | 7.2.6.1 | Update `internal/tui/explorer.go` -- bookmark toggle: Handle `b` key: toggle bookmark on the currently highlighted row. Send a `tea.Cmd` that calls `store.SetBookmark(ctx, domain, !currentBookmarkState)`. On `BookmarkToggleMsg` received back, update the local hit's `Bookmarked` field and refresh the table row. Add a visual indicator: prepend a star character (e.g., `*` in gold/yellow via `StyleBookmarked`) to the Domain column for bookmarked hits. | CLI |
| ⬜ | 7.2.6.2 | Update `internal/tui/filter.go` -- bookmark filter: Add a "Bookmarked" toggle field to the filter overlay. Cycle with left/right arrows between "" (all), "yes" (bookmarked only), "no" (unbookmarked only). Map to `QueryFilter.Bookmarked` in `buildFilter`. Display in the explorer filter bar: `bookmarked:yes` when active. | CLI |
| ⬜ | 7.2.6.3 | Update `internal/tui/explorer.go` -- live domain indicator: In `hitsToRows`, if a hit has `IsLive == true`, render the domain text with `StyleLiveDomain` (green). In the detail view this is already handled by 7.2.4.6. In the explorer table, add a small `[L]` tag next to live domains or color the entire domain cell green. | CLI |
| ⬜ | 7.2.6.4 | **BUILD CHECK** -- `go build ./internal/tui/... && go vet ./internal/tui/...` passes clean. | CLI |

#### 7.2.7 Backtrack & Enrichment CLI Flags (CLI) [worktree]

| Status | Task | Description | Agent |
|--------|------|-------------|-------|
| ⬜ | 7.2.7.1 | Update `internal/cmd/watch.go`: Add `--backtrack` flag (int64, default 0) to the watch command. Pass the backtrack value through to the poller manager via config. Add flag description: "Start N entries behind the current log tip for immediate results (default: 0, start at tip)". Also wire the enrichment pipeline: after pollers start, create an `Enricher`, start it, and connect it to receive domains from the hit channel (tap the hit channel so enricher gets a copy of each scored domain). | CLI |
| ⬜ | 7.2.7.2 | Update `internal/cmd/query.go`: Add `--bookmarked` flag (bool, default false). Add `--live-only` flag (bool, default false). Map both to the `QueryFilter` fields. Update the help text to document the new flags. | CLI |
| ⬜ | 7.2.7.3 | Update `internal/cmd/output.go`: Extend `FormatTable` to show a bookmark indicator (`*` prefix) and live indicator (`[L]` suffix) on applicable rows. Extend `FormatJSON` and `FormatCSV` to include the new fields (is_live, resolved_ips, hosting_provider, http_status, bookmarked) in output. | CLI |
| ⬜ | 7.2.7.4 | **BUILD CHECK** -- `go build ./cmd/ctsnare && go vet ./internal/cmd/... && golangci-lint run ./internal/cmd/...` passes clean. `./ctsnare watch --help` shows --backtrack flag. `./ctsnare query --help` shows --bookmarked and --live-only flags. | CLI |

---

### Merge Gate 3: Post Phase 7.2

> All parallel work from Phase 7.2 STOPS here. No agent continues until merge is clean.

#### Prerequisites
- ⬜ BE (backend-engineer) has committed and pushed their worktree branch
- ⬜ CLI (cli-engineer) has committed and pushed their worktree branch
- ⬜ BE build verification passes in their worktree
- ⬜ CLI build verification passes in their worktree

#### Merge Protocol
1. Lead session creates fresh branch or works on main
2. Merge BE branch into main -- resolve any conflicts
3. Merge CLI branch into main -- resolve any conflicts
4. Run `go mod tidy` to reconcile any dependency differences (enrichment may need `golang.org/x/time` if rate limiter used)
5. Run full verification: `go build -o ctsnare ./cmd/ctsnare && go vet ./... && golangci-lint run ./... && go test ./...`
6. Fix any issues before proceeding (failing agent fixes their code)
7. All agents pull fresh main before Phase 7.3

#### Conflict Resolution Priority
1. `internal/domain/` -- canonical source of truth (should have no conflicts -- frozen after Phase 7.1)
2. `go.mod` / `go.sum` -- run `go mod tidy` on merged result
3. `internal/tui/messages.go`, `internal/tui/styles.go`, `internal/tui/keys.go` -- frozen after 7.1, no conflicts expected
4. `cmd/ctsnare/main.go` -- CLI agent's version takes precedence

#### Post-Merge Verification
- ⬜ `go build -o ctsnare ./cmd/ctsnare` succeeds
- ⬜ `go vet ./...` reports zero issues
- ⬜ `golangci-lint run ./...` reports zero issues
- ⬜ `go test ./...` all tests pass
- ⬜ `./ctsnare watch --help` shows --backtrack flag
- ⬜ `./ctsnare query --help` shows --bookmarked and --live-only flags

---

### Phase 7.3: Enhancement Integration (Sequential)

> Sequential. CLI agent (or lead) wires enrichment into TUI, connects discarded domain feed,
> and ensures all new features work end-to-end.
> Depends on: Merge Gate 3 (must pass clean)
>
> **Risk: MEDIUM** -- This is where the enrichment pipeline meets the TUI. Interface mismatches
> between the enrichment results and TUI display surface here. Fix them immediately.

#### File Ownership: CLI agent owns all files. BE agent available for consultation on enrichment/storage issues.

#### 7.3.1 Wire Enrichment into Watch Command

| Status | Task | Description | Agent |
|--------|------|-------------|-------|
| ⬜ | 7.3.1.1 | Update `internal/cmd/watch.go` `runTUI`: Create the enrichment channel (`enrichChan chan enrichment.EnrichResult`, buffered 256). Create the `Enricher` with store and enrichChan. Start the enricher goroutine (`enricher.Run(ctx)`). After each hit is sent to `hitChan`, also call `enricher.Enqueue(hit.Domain)` so every scored domain gets probed. Bridge the `enrichChan` to the TUI by creating a new `waitForEnrichment` tea.Cmd in the app model that reads from the channel and converts `EnrichResult` to `EnrichmentMsg`. | CLI |
| ⬜ | 7.3.1.2 | Update `internal/cmd/watch.go` `runTUI`: Wire the discarded domain feed. In the poller, when a domain scores zero, send a `DiscardedDomainMsg` to the TUI. This requires adding a `discardChan chan<- string` parameter to the poller (or using the existing stats channel to piggyback discarded domain names). The simplest approach: add a `discardChan chan string` (buffered 256) alongside hitChan, pass it to the poller manager, and have pollers send discarded domains on it. In the app model, add a `waitForDiscard` tea.Cmd that reads from discardChan and sends `DiscardedDomainMsg`. | CLI |
| ⬜ | 7.3.1.3 | Update `internal/tui/app.go`: Handle `EnrichmentMsg` in Update -- find the matching hit in the feed's hit buffer by domain name and update its enrichment fields (IsLive, ResolvedIPs, etc.). If the explorer is showing that hit, trigger a refresh. Handle `DiscardedDomainMsg` in Update -- forward to FeedModel. Handle `BookmarkToggleMsg` -- forward to ExplorerModel to refresh the affected row. Handle `DeleteHitsMsg` -- forward to ExplorerModel to reload hits from DB. Add `enrichChan` and `discardChan` fields to AppModel. Wire `waitForEnrichment` and `waitForDiscard` in `Init()`. | CLI |
| ⬜ | 7.3.1.4 | Update `internal/cmd/watch.go` `runHeadless`: Wire enrichment in headless mode too. Create enricher, start it, enqueue domains from hitChan. Enrichment results are written to DB silently. Drain enrichChan in a background goroutine. Also wire the discardChan drain in headless mode. | CLI |
| ⬜ | 7.3.1.5 | **BUILD CHECK** -- `go build -o ctsnare ./cmd/ctsnare && go vet ./... && golangci-lint run ./... && go test ./...` passes clean. | CLI |

#### 7.3.2 End-to-End Integration Smoke Test

| Status | Task | Description | Agent |
|--------|------|-------------|-------|
| ⬜ | 7.3.2.1 | Update `internal/cmd/integration_test.go`: Add integration tests for the new features: (1) Test backtrack mode -- start headless with --backtrack 100 against a mock CT log, verify it processes entries from tree_size-100. (2) Test bookmark workflow -- insert hits, bookmark one via store, query with --bookmarked, verify only bookmarked hit returned. (3) Test delete workflow -- insert hits, delete one, verify it is gone from query results. (4) Test enrichment fields in query output -- insert a hit with enrichment data, query as JSON, verify enrichment fields present. | CLI |
| ⬜ | 7.3.2.2 | Update `internal/cmd/integration_test.go`: Test query with new flags: `--live-only` returns only hits where is_live=true. `--bookmarked` returns only bookmarked hits. Both flags together compose correctly. Test `db export --format jsonl` includes enrichment fields. Test `db export --format csv` includes enrichment column headers. | CLI |
| ⬜ | 7.3.2.3 | **BUILD CHECK** -- `go build -o ctsnare ./cmd/ctsnare && go vet ./... && golangci-lint run ./... && go test -count=1 ./...` all pass. Manual smoke test: `./ctsnare watch --headless --backtrack 1000 --verbose` starts, processes backlogged entries, and enrichment probes run in background. Ctrl-C shuts down cleanly. | CLI |

---

### Phase 7.4: Enhancement Polish -- PARALLEL

> PARALLEL. Two agents work simultaneously on test coverage and final TUI polish.
> Depends on: Phase 7.3 (all tasks complete, full build passes clean)
>
> **Risk: LOW** -- QA adds test files only. CLI does final visual polish on TUI files it already owns.
> No new shared file modifications.

#### File Ownership (This Phase)

| Agent | Owns (read/write) | Can Read (no write) |
|-------|-------------------|-------------------|
| QA (qa-engineer) | `*_test.go` files in all packages | All source files (read only) |
| CLI (cli-engineer) | `internal/tui/` (visual polish only -- no new features) | All source files (read only) |

**OFF LIMITS during this phase:** `go.mod`, `go.sum`, all non-test `.go` files (except `internal/tui/` for CLI), `.golangci.yml`, `internal/domain/`, `internal/enrichment/`, `internal/storage/`, `internal/poller/`, `internal/config/`

#### 7.4.1 Test Coverage for Phase 7 Features (QA) [worktree]

| Status | Task | Description | Agent |
|--------|------|-------------|-------|
| ⬜ | 7.4.1.1 | Create `internal/enrichment/enricher_integration_test.go`. Integration test using httptest.NewServer as a live domain and a non-routable address as dead domain. Verify full pipeline: enqueue domain, enricher probes it, store.UpdateEnrichment is called, enrichChan receives result. Verify rate limiting works end-to-end (5 concurrent, 1/sec rate). Verify graceful shutdown clears in-flight probes. | QA |
| ⬜ | 7.4.1.2 | Expand `internal/storage/db_test.go` with Phase 7 edge cases: test bookmark toggle (set, unset, set again). Test DeleteHits with empty domain list (no-op). Test DeleteHits with domains that don't exist (no error). Test UpdateEnrichment for a domain that doesn't exist (no error or specific error). Test QueryHits with both Bookmarked and LiveOnly set simultaneously. Test QueryHits sorting by new columns (is_live, bookmarked, http_status). | QA |
| ⬜ | 7.4.1.3 | Create `internal/poller/backtrack_test.go`. Dedicated tests for backtrack behavior: mock CT log with tree_size=10000, test backtrack=5000 starts at 5000, test backtrack=0 starts at 10000, test backtrack=20000 (exceeds tree_size) starts at 0, test backtrack with changing tree_size (initial STH returns 10000, poller starts at 5000, next STH returns 12000 -- poller should process 5000-12000). | QA |
| ⬜ | 7.4.1.4 | Expand `internal/tui/app_test.go` with Phase 7 message handling: test EnrichmentMsg updates feed model hit's enrichment fields, test DiscardedDomainMsg arrives and is forwarded to feed, test BookmarkToggleMsg refreshes explorer, test DeleteHitsMsg triggers explorer reload. | QA |
| ⬜ | 7.4.1.5 | **BUILD CHECK** -- `go test -v -count=1 -race ./...` all pass with zero failures. Run `go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out` and verify enrichment package has >70% coverage. | QA |

#### 7.4.2 TUI Final Polish (CLI) [worktree]

| Status | Task | Description | Agent |
|--------|------|-------------|-------|
| ⬜ | 7.4.2.1 | Polish `internal/tui/feed.go` -- tune the auto-scrolling behavior: Verify discarded domains fade correctly after 2 seconds. Ensure the viewport auto-scrolls to show the newest entries (viewport offset tracks the top of the content). When the user has manually scrolled up, disable auto-scroll until they scroll back to the bottom (so they can read old entries without being yanked). Add a `[LIVE]` indicator in the header when auto-scroll is active, `[PAUSED]` when manual scroll is active. | CLI |
| ⬜ | 7.4.2.2 | Polish `internal/tui/explorer.go` -- selection UX refinement: Ensure multi-select state survives sort changes and filter application (clear selection on filter change, preserve on sort). Show selection count in the filter bar: "3 selected" when items are selected. Verify delete confirmation works correctly with the confirmation overlay and doesn't swallow subsequent key presses. | CLI |
| ⬜ | 7.4.2.3 | Polish `internal/tui/feed.go` and `internal/tui/explorer.go` -- help bar consistency: Ensure both views have a consistent help bar at the bottom. Feed view: `Tab=views | q=quit | ?=help | j/k=scroll`. Explorer view: `Tab=views | q=quit | s=sort | f=filter | b=mark | Space=select | d=delete | Enter=detail`. Detail view: `Esc=back | j/k=scroll`. The help bar should always show the keybindings relevant to the current view. | CLI |
| ⬜ | 7.4.2.4 | Polish all TUI views -- overall consistency pass: Verify severity colors are consistent across feed, explorer, and detail views. Verify live domain highlighting (green) appears in all three views. Verify bookmark star appears in explorer and detail views. Verify the status bar width fills the terminal. Test with narrow terminals (80 cols) and wide terminals (200+ cols) to ensure layout degrades gracefully. | CLI |
| ⬜ | 7.4.2.5 | **BUILD CHECK** -- `go build ./internal/tui/... && go vet ./internal/tui/... && go test ./internal/tui/...` passes clean. | CLI |

---

### Merge Gate 4: Post Phase 7.4

> All parallel work from Phase 7.4 STOPS here. No agent continues until merge is clean.

#### Prerequisites
- ⬜ QA has committed and pushed their worktree branch (test files only)
- ⬜ CLI has committed and pushed their worktree branch (TUI polish only)
- ⬜ QA build verification passes in their worktree
- ⬜ CLI build verification passes in their worktree

#### Merge Protocol
1. Lead session merges QA branch into main (test files only -- should be conflict-free)
2. Lead session merges CLI branch into main (TUI files only -- should be conflict-free)
3. Run `go mod tidy` if any test dependencies were added
4. Run full verification: `make check` (or equivalent: build + vet + lint + test)
5. Fix any issues before proceeding
6. Tag as `v0.6.0` enhancement milestone

#### Conflict Resolution Priority
1. Test files -- should never conflict (QA is the only agent writing tests)
2. TUI files -- should never conflict (CLI is the only agent modifying TUI in this phase)
3. `go.mod` / `go.sum` -- run `go mod tidy` on merged result

#### Post-Merge Verification
- ⬜ `make check` passes (build + vet + lint + test with race detection)
- ⬜ All new tests from QA pass (zero data races)
- ⬜ `./ctsnare watch --help` shows all new flags (--backtrack)
- ⬜ `./ctsnare query --help` shows all new flags (--bookmarked, --live-only)
- ⬜ `make build` produces working binary
- ⬜ Manual smoke test: TUI severity colors, auto-scrolling feed, batch delete, bookmarks, enrichment display all work

---

## Parallelization Map

```
Phase 1: Foundation (Sequential, BE)
|-- 1.1 Scaffold -----> 1.2 Dependencies -----> 1.3 Core Types -----> 1.4 Cobra Root
|
|============== All Phase 1 tasks complete, build passes clean ===============
|
Phase 2: Core Engine (PARALLEL, worktrees)
|
|-- BE (backend-engineer) worktree:            CLI (cli-engineer) worktree:
|   |                                          |
|   |-- 2.1 Config System                      |-- 2.6 TUI Styles & Layout
|   |-- 2.2 Keyword Profiles                   |-- 2.7 TUI Live Feed View
|   |-- 2.3 Scoring Engine                     |-- 2.8 TUI DB Explorer View
|   |-- 2.4 SQLite Storage                     |-- 2.9 TUI Root App
|   |-- 2.5 CT Log Poller                      |-- 2.10 Cobra Subcommands
|   |                                          |
|   v                                          v
|
|================ MERGE GATE 1 (Lead merges, full build) ====================
|
Phase 3: Integration (Sequential, CLI)
|-- 3.1 Wire Watch Command
|-- 3.2 Graceful Shutdown & Signals
|-- 3.3 End-to-End Smoke Test
|
|============== All Phase 3 tasks complete, build passes clean ===============
|
Phase 4: Hardening (PARALLEL, worktrees)
|
|-- QA worktree:           SEC worktree:           OPS worktree:
|   |                      |                       |
|   |-- 4.1 Test Coverage  |-- 4.2 Security Audit  |-- 4.3 CI/CD & Build
|   |                      |                       |
|   v                      v                       v
|
|================ MERGE GATE 2 (Lead merges, full build) ====================
|
Phase 5: Polish & Release (Sequential, BE/DOC)
|-- 5.1 Security Remediation
|-- 5.2 Internal Documentation (CLAUDE.md)
|
|============== All Phase 5 tasks complete, build passes clean ===============
|
Phase 6: Documentation (Sequential, DOC)
|-- 6.1 README ---------> 6.2 Go Doc Comments ---------> 6.3 Help Text & CLI Polish
|                                                                     |
|                                                                     v
|                                                         6.4 Changelog & Release Milestone
|
|============== All Phase 6 tasks complete, build passes clean ===============
|
Phase 7: Enhancements
|
|-- Phase 7.1: Enhancement Foundation (Sequential, BE)
|   |
|   |-- 7.1.1 Domain Type Extensions
|   |-- 7.1.2 Storage Schema Migration
|   |-- 7.1.3 TUI Message Types & Shared Contracts
|   |-- 7.1.4 Enrichment Package Scaffold
|   |
|   | ** Domain types REFROZEN after 7.1 **
|   |
|   |============== All Phase 7.1 tasks complete, build passes clean ===========
|   |
|   Phase 7.2: Enhancement Build (PARALLEL, worktrees)
|   |
|   |-- BE (backend-engineer) worktree:        CLI (cli-engineer) worktree:
|   |   |                                      |
|   |   |-- 7.2.1 Enrichment Pipeline          |-- 7.2.4 TUI Visual Overhaul
|   |   |-- 7.2.2 Backtrack Mode               |-- 7.2.5 Batch Delete & Selection
|   |   |-- 7.2.3 Storage & Export Extensions   |-- 7.2.6 Bookmark/Flag System TUI
|   |   |                                      |-- 7.2.7 Backtrack & Enrichment CLI
|   |   |                                      |
|   |   v                                      v
|   |
|   |================ MERGE GATE 3 (Lead merges, full build) ==================
|   |
|   Phase 7.3: Enhancement Integration (Sequential, CLI)
|   |-- 7.3.1 Wire Enrichment into Watch Command
|   |-- 7.3.2 End-to-End Integration Smoke Test
|   |
|   |============== All Phase 7.3 tasks complete, build passes clean ===========
|   |
|   Phase 7.4: Enhancement Polish (PARALLEL, worktrees)
|   |
|   |-- QA (qa-engineer) worktree:             CLI (cli-engineer) worktree:
|   |   |                                      |
|   |   |-- 7.4.1 Test Coverage                |-- 7.4.2 TUI Final Polish
|   |   |                                      |
|   |   v                                      v
|   |
|   |================ MERGE GATE 4 (Lead merges, full build) ==================
|   |
|   v0.6.0 Enhancement Milestone
```

---

## Changelog Reference

See `.project/changelog.md` for detailed version history.

---

## Notes & Decisions

### Architecture Decisions
- **Domain types frozen after Phase 1, briefly unfrozen in Phase 7.1**: All agents build against `internal/domain/` types established in Phase 1. Types were frozen through Phases 2-6. Phase 7.1 temporarily unfreezes them to add enrichment, bookmark, and liveness fields. After Phase 7.1, types are REFROZEN for Phase 7.2+ parallel work. This controlled unfreeze minimizes ripple effects.
- **CLI agent owns all Cobra commands**: Even though the watch command wires backend components, the CLI agent owns the command files. This avoids two agents modifying the same file. The integration wiring happens in Phase 3 (sequential) to avoid conflicts.
- **Storage interface for TUI decoupling**: The TUI never calls storage directly during Phase 2. It receives hits via channels and uses the `Store` interface only in Phase 3 when the explorer view is wired. This allows the CLI agent to build TUI without a working database.
- **Separate export from storage query**: Export functions (JSONL, CSV) live in storage package but use QueryHits internally. This keeps the export logic close to the data and avoids a separate export package.

### Dependency Management
- All `go get` commands run in Phase 1 only. No new dependencies added during parallel phases.
- If a dependency is discovered to be needed during Phase 2, the agent notes it and it's added at Merge Gate 1.
- Phase 7 may require `golang.org/x/time/rate` for the enrichment rate limiter -- add at Merge Gate 3 if needed.
- `go mod tidy` runs at every merge gate to keep go.sum clean.

### Known Risks
- **CT log shard discovery**: The PRD mentions automatic discovery of active shards. For v1, hard-code current shard URLs in default config. Shard rotation is a future enhancement.
- **Certificate parsing edge cases**: Some CT log entries contain pre-certificates, redacted certificates, or non-standard extensions. The parser should handle these gracefully (skip with warning, never panic).
- **SQLite concurrent write performance**: WAL mode handles concurrent reads well but writes are serialized. The poller manager's multiple goroutines all write through a single DB handle. This should be fine for expected throughput (hundreds of hits/sec) but could become a bottleneck at scale.
- **Enrichment probe rate limiting (Phase 7)**: The liveness probe makes HTTP requests to potentially malicious domains. Rate limiting is critical -- 5 concurrent / 1 req/sec prevents abuse. The enricher must never block the polling pipeline; it runs independently and writes back to the DB.
- **Schema migration backward compatibility (Phase 7)**: Adding columns to an existing SQLite table requires idempotent ALTER TABLE statements. The migration must handle the case where the DB already has the new columns (e.g., from a previous run) without erroring.
- **Domain type unfreeze risk (Phase 7.1)**: Unfreezing domain types is the highest-risk operation in Phase 7. All new fields use zero-value defaults and all new interface methods are additive. Existing code paths must continue to work without modification -- new fields are opt-in.

### Conflict Zone Incidents
- [None yet -- log any merge conflicts or agent collisions here for future reference]

---

*Last updated: 2026-02-24*
*Current Phase: Phase 7.1 -- Enhancement Foundation (pending)*
*Previous Milestone: v0.5.0 Feature Complete -- README, Go doc comments, CLI help text, changelog finalized*
*Next Milestone: v0.6.0 Enhancement Complete -- enrichment pipeline, TUI overhaul, backtrack mode, batch delete, bookmarks*
