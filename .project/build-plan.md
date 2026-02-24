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

---

## Build Verification Protocol

> How and when builds are verified during parallel execution.

| Context | What to Run | Who Runs It |
|---------|------------|-------------|
| During Phase 2 -- BE agent | `go build ./internal/poller/... ./internal/scoring/... ./internal/profile/... ./internal/storage/... ./internal/config/... && go vet ./internal/poller/... ./internal/scoring/... ./internal/profile/... ./internal/storage/... ./internal/config/... && go test ./internal/poller/... ./internal/scoring/... ./internal/profile/... ./internal/storage/... ./internal/config/...` | BE in worktree |
| During Phase 2 -- CLI agent | `go build ./internal/tui/... ./internal/cmd/... ./cmd/ctsnare/... && go vet ./internal/tui/... ./internal/cmd/... ./cmd/ctsnare/... && go test ./internal/tui/... ./internal/cmd/...` | CLI in worktree |
| During Phase 4 -- QA agent | `go test -v -count=1 ./...` | QA in worktree |
| During Phase 4 -- OPS agent | `golangci-lint run ./... && go build -o ctsnare ./cmd/ctsnare` | OPS in worktree |
| At merge gates | `go build -o ctsnare ./cmd/ctsnare && go vet ./... && golangci-lint run ./... && go test ./...` | Lead session on merged main |
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
Phase 3: Integration         [░░░░░░░░░░░░░░░░░░░░]   0%  ⬜
Phase 4: Hardening           [░░░░░░░░░░░░░░░░░░░░]   0%  ⬜
  🔀 Merge Gate 2            [░░░░░░░░░░░░░░░░░░░░]   0%  ⬜
Phase 5: Polish & Release    [░░░░░░░░░░░░░░░░░░░░]   0%  ⬜
─────────────────────────────────────────────────────────────
Overall Progress             [███████████░░░░░░░░░]  57%
```

| Phase | Tasks | Completed | Blocked | Deferred | Progress | Agents |
|-------|-------|-----------|---------|----------|----------|--------|
| Phase 1: Foundation | 15 | 15 | 0 | 0 | 100% | BE |
| Phase 2: Core Engine | 28 | 28 | 0 | 0 | 100% | BE, CLI |
| Merge Gate 1 | 1 | 1 | 0 | 0 | 100% | Lead |
| Phase 3: Integration | 10 | 0 | 0 | 0 | 0% | CLI |
| Phase 4: Hardening | 15 | 0 | 0 | 0 | 0% | QA, SEC, OPS |
| Merge Gate 2 | 1 | 0 | 0 | 0 | 0% | Lead |
| Phase 5: Polish & Release | 6 | 0 | 0 | 0 | 0% | BE, CLI, DOC |
| **Total** | **76** | **44** | **0** | **0** | **57%** | |

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
- [ ] BE (backend-engineer) has committed and pushed their worktree branch
- [ ] CLI (cli-engineer) has committed and pushed their worktree branch
- [ ] BE build verification passes in their worktree
- [ ] CLI build verification passes in their worktree

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
- [ ] `go build -o ctsnare ./cmd/ctsnare` succeeds
- [ ] `go vet ./...` reports zero issues
- [ ] `golangci-lint run ./...` reports zero issues
- [ ] `go test ./...` all tests pass
- [ ] `./ctsnare --help` shows all subcommands
- [ ] `./ctsnare watch --help` shows watch flags
- [ ] `./ctsnare query --help` shows query flags

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
| ⬜ | 3.1.1 | Update `internal/cmd/watch.go` RunE to wire real components: load config (internal/config), open storage (internal/storage), create scoring engine (internal/scoring), load profile (internal/profile), create hit and stats channels (buffered), create poller manager and start it, create TUI app model with store + channels + profile name, run `tea.NewProgram(app, tea.WithAltScreen())`. For headless mode: skip TUI, just run poller manager, block on context until SIGINT/SIGTERM. Handle graceful shutdown: context cancellation stops pollers, close DB on exit. | CLI |
| ⬜ | 3.1.2 | Update `internal/cmd/query.go` RunE to wire real storage: load config for DB path, open storage, build QueryFilter from flags, call store.QueryHits, format output, close DB. Handle edge cases: no results found (print message, not error), DB file doesn't exist (friendly error message). | CLI |
| ⬜ | 3.1.3 | Update `internal/cmd/db.go` subcommands to wire real storage: each subcommand opens DB via config, performs operation, closes DB. `db path` reads from config. `db export` writes to file or stdout based on --output flag. | CLI |
| ⬜ | 3.1.4 | Update `internal/cmd/profiles.go` to wire real profile manager: create manager with config's custom profiles, list/show operations work against real data. | CLI |
| ⬜ | 3.1.5 | **BUILD CHECK** -- `go build -o ctsnare ./cmd/ctsnare && go vet ./... && golangci-lint run ./... && go test ./...` passes clean. | CLI |

### 3.2 Graceful Shutdown & Signal Handling

| Status | Task | Description | Agent |
|--------|------|-------------|-------|
| ⬜ | 3.2.1 | Implement signal handling in watch command: listen for SIGINT and SIGTERM, cancel context, wait for pollers to drain (with timeout), close database, restore terminal (bubbletea handles this for TUI mode). Log shutdown progress with slog. For headless mode, use `signal.NotifyContext`. | CLI |
| ⬜ | 3.2.2 | Add structured logging throughout: use `log/slog` with JSON handler when --verbose is set, text handler otherwise. Log at appropriate levels: Info for startup/shutdown events, Debug for per-entry processing, Warn for skipped/malformed entries, Error for failures. Add slog initialization in root command PersistentPreRunE. | CLI |
| ⬜ | 3.2.3 | **BUILD CHECK** -- `go build -o ctsnare ./cmd/ctsnare && go vet ./... && golangci-lint run ./... && go test ./...` passes clean. Manual smoke test: `./ctsnare watch --headless --verbose` starts without error, Ctrl-C shuts down cleanly. | CLI |

### 3.3 End-to-End Smoke Test

| Status | Task | Description | Agent |
|--------|------|-------------|-------|
| ⬜ | 3.3.1 | Create `internal/cmd/integration_test.go`. Integration test: create temp DB, create config pointing to it, start watch in headless mode with a short timeout context (5 seconds), verify it starts and stops cleanly. Test query command with pre-populated DB (insert test hits, run query, verify output format). Test db stats with known data. Test profiles list output. | CLI |
| ⬜ | 3.3.2 | **BUILD CHECK** -- `go build -o ctsnare ./cmd/ctsnare && go vet ./... && golangci-lint run ./... && go test -count=1 ./...` all pass. Binary runs and responds to --help, watch --help, query --help, db --help, profiles --help. | CLI |

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
| ⬜ | 4.1.1 | Create `internal/scoring/heuristics_test.go`. Exhaustive table-driven tests for each heuristic function: keyword matching (case variations, partial matches, no match, empty input), TLD scoring (exact TLD match, subdomain of TLD, no match), domain length (exactly 30 chars, 31 chars, short domain), hyphen density (0, 1, 2, 3+ hyphens), number sequences (3 digits no score, 4 digits scores, 10 digits scores), multi-keyword bonus (0, 1, 2, 3, 5 keywords). | QA |
| ⬜ | 4.1.2 | Create `internal/poller/manager_test.go`. Test manager starts and stops cleanly, test context cancellation stops all pollers, test stats aggregation from multiple pollers. Use mocked CT log server (httptest). | QA |
| ⬜ | 4.1.3 | Create `internal/config/defaults_test.go`. Test that DefaultConfig returns valid config with all required fields populated: at least one CT log URL, valid batch size (>0), valid poll interval (>0), valid DB path, non-empty skip suffixes. Test XDG path construction. | QA |
| ⬜ | 4.1.4 | Expand `internal/storage/db_test.go` with edge cases: concurrent read/write (goroutines inserting while querying), very long domain names, Unicode domains, empty keyword arrays, null-like fields, QueryFilter with all fields set simultaneously, pagination (limit + offset), sort by each column. | QA |
| ⬜ | 4.1.5 | Create `internal/tui/feed_test.go`. Test FeedModel: hit buffer doesn't exceed max size, new hits prepend correctly, stats update correctly, viewport resize recalculates, severity styling applies correctly (unit test the View output for known hits). | QA |
| ⬜ | 4.1.6 | **BUILD CHECK** -- `go test -v -count=1 -race ./...` all pass with zero failures. Run `go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out` and verify coverage is reported. | QA |

### 4.2 Security Audit (SEC) [worktree] [cloud]

| Status | Task | Description | Agent |
|--------|------|-------------|-------|
| ⬜ | 4.2.1 | Audit `internal/storage/` for SQL injection: verify all queries use parameterized placeholders, no string concatenation for SQL, no fmt.Sprintf in query construction. Verify QueryHits dynamic query builder is safe. File issues for any findings. | SEC |
| ⬜ | 4.2.2 | Audit `internal/poller/` for SSRF and input validation: verify CT log URLs are validated, verify certificate parsing handles malformed input without panic, verify HTTP client has timeouts set, verify no user-controlled URLs are fetched. File issues for any findings. | SEC |
| ⬜ | 4.2.3 | Audit `internal/config/` for path traversal: verify DB path is sanitized, verify config file path is validated, verify XDG directory creation uses safe permissions (0700 for dirs, 0600 for files). Audit TOML parsing for denial of service (deeply nested, huge files). File issues for any findings. | SEC |
| ⬜ | 4.2.4 | Run `govulncheck ./...` to check for known vulnerabilities in dependencies. Run `go mod verify` to check module integrity. File issues for any findings. | SEC |
| ⬜ | 4.2.5 | **AUDIT REPORT** -- Write security assessment summary to `.project/issues/open/ISSUE-001-security-audit-v1.md` using the security report template. Include findings, risk ratings, and remediation recommendations. | SEC |

### 4.3 CI/CD & Build Infrastructure (OPS) [worktree] [cloud]

| Status | Task | Description | Agent |
|--------|------|-------------|-------|
| ⬜ | 4.3.1 | Create `Makefile` with targets: `build` (go build -o ctsnare ./cmd/ctsnare), `test` (go test -race -count=1 ./...), `lint` (golangci-lint run ./...), `fmt` (gofmt -w .), `vet` (go vet ./...), `clean` (rm -f ctsnare *.db coverage.out), `coverage` (go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out), `check` (build + vet + lint + test -- full CI suite), `run` (go run ./cmd/ctsnare), `help` (print all targets). All targets use `.PHONY`. | OPS |
| ⬜ | 4.3.2 | Create `.github/workflows/ci.yml`: trigger on push to main and pull_request. Jobs: `lint` (runs golangci-lint via golangci-lint-action), `test` (runs go test -race -count=1 ./... on ubuntu-latest with Go 1.26), `build` (runs go build, uploads binary as artifact). Cache Go modules. Matrix: test on ubuntu-latest. | OPS |
| ⬜ | 4.3.3 | Create `.github/workflows/release.yml`: trigger on tag push (v*). Uses goreleaser to cross-compile for linux-amd64, linux-arm64, darwin-amd64, darwin-arm64, windows-amd64. Creates GitHub release with binaries and checksums. | OPS |
| ⬜ | 4.3.4 | Create `.goreleaser.yml`: project name ctsnare, builds from ./cmd/ctsnare, GOOS/GOARCH matrix, archive format tar.gz (linux/mac) and zip (windows), checksum file, changelog from git log. | OPS |
| ⬜ | 4.3.5 | **BUILD CHECK** -- `make check` passes clean (build + vet + lint + test). Verify CI workflow YAML is valid. | OPS |

---

## Merge Gate 2: Post Phase 4

> All parallel work from Phase 4 STOPS here. No agent continues until merge is clean.

### Prerequisites
- [ ] QA has committed and pushed their worktree branch (new test files only)
- [ ] SEC has committed audit report to `.project/issues/open/`
- [ ] OPS has committed and pushed their worktree branch (Makefile, CI/CD, Dockerfile, goreleaser)
- [ ] QA build verification passes in their worktree
- [ ] OPS build verification passes in their worktree

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
- [ ] `make check` passes (build + vet + lint + test)
- [ ] All new tests from QA pass
- [ ] `make build` produces working binary
- [ ] CI workflow files are present and valid YAML
- [ ] Security audit report is present in `.project/issues/open/`

---

## Phase 5: Polish & Release Preparation

> Sequential. Final polish, documentation, and release prep.
> Depends on: Merge Gate 2 (must pass clean) + critical security findings addressed
>
> **Risk: LOW** -- Polish and documentation. Core functionality is complete and tested.

### File Ownership: Lead session (BE/CLI) owns all files.

### 5.1 Security Remediation

| Status | Task | Description | Agent |
|--------|------|-------------|-------|
| ⬜ | 5.1.1 | Address all CRITICAL and HIGH findings from the Phase 4 security audit. Implement fixes as described in the issue files. Mark issues as resolved. Run full test suite after each fix. | BE |
| ⬜ | 5.1.2 | **BUILD CHECK** -- `make check` passes clean after all security fixes. | BE |

### 5.2 Documentation & Final Polish

| Status | Task | Description | Agent |
|--------|------|-------------|-------|
| ⬜ | 5.2.1 | Update `/.claude/CLAUDE.md` with final project structure, build commands, and any new conventions established during development. Fill in all placeholder sections. | DOC |
| ⬜ | 5.2.2 | Create `README.md` at project root: project name + one-liner, installation instructions (go install + binary download), quick start (ctsnare watch, ctsnare query), all subcommands with examples, configuration section (TOML config file location, all configurable options with defaults), built-in profiles list, architecture overview (data flow diagram from PRD), development section (make dev, make test, make lint), license. | DOC |
| ⬜ | 5.2.3 | Add Go doc comments to all exported functions and types across all packages. Every exported symbol in `internal/domain/`, `internal/config/`, `internal/storage/`, `internal/scoring/`, `internal/profile/`, `internal/poller/`, `internal/tui/`, `internal/cmd/` gets a doc comment following Go conventions (starts with the name of the symbol). | DOC |
| ⬜ | 5.2.4 | Update `.project/changelog.md` with all changes from v0.1.0 (project setup through feature complete). Mark milestone 0.5.0 as Feature Complete with today's date. | DOC |

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
Phase 5: Polish & Release (Sequential, BE/CLI/DOC)
|-- 5.1 Security Remediation
|-- 5.2 Documentation & Final Polish
```

---

## Changelog Reference

See `.project/changelog.md` for detailed version history.

---

## Notes & Decisions

### Architecture Decisions
- **Domain types frozen after Phase 1**: All agents build against `internal/domain/` types established in Phase 1. Changes to domain types after Phase 1 require a sequential phase and all agents must re-sync. This prevents interface mismatch bugs.
- **CLI agent owns all Cobra commands**: Even though the watch command wires backend components, the CLI agent owns the command files. This avoids two agents modifying the same file. The integration wiring happens in Phase 3 (sequential) to avoid conflicts.
- **Storage interface for TUI decoupling**: The TUI never calls storage directly during Phase 2. It receives hits via channels and uses the `Store` interface only in Phase 3 when the explorer view is wired. This allows the CLI agent to build TUI without a working database.
- **Separate export from storage query**: Export functions (JSONL, CSV) live in storage package but use QueryHits internally. This keeps the export logic close to the data and avoids a separate export package.

### Dependency Management
- All `go get` commands run in Phase 1 only. No new dependencies added during parallel phases.
- If a dependency is discovered to be needed during Phase 2, the agent notes it and it's added at Merge Gate 1.
- `go mod tidy` runs at every merge gate to keep go.sum clean.

### Known Risks
- **CT log shard discovery**: The PRD mentions automatic discovery of active shards. For v1, hard-code current shard URLs in default config. Shard rotation is a future enhancement.
- **Certificate parsing edge cases**: Some CT log entries contain pre-certificates, redacted certificates, or non-standard extensions. The parser should handle these gracefully (skip with warning, never panic).
- **SQLite concurrent write performance**: WAL mode handles concurrent reads well but writes are serialized. The poller manager's multiple goroutines all write through a single DB handle. This should be fine for expected throughput (hundreds of hits/sec) but could become a bottleneck at scale.

### Conflict Zone Incidents
- [None yet -- log any merge conflicts or agent collisions here for future reference]

---

*Last updated: 2026-02-24*
*Current Phase: Merge Gate 1 COMPLETE -- ready for Phase 3*
*Next Milestone: Phase 3 Integration (sequential, CLI agent)*
