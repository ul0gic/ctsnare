# ctsnare

> Go CLI tool that monitors Certificate Transparency logs in real-time, scores domains against keyword profiles, stores hits in embedded SQLite, and provides a TUI dashboard + CLI query interface.

## Project Documentation

Read these before making changes:

- @.project/prd.md — Product requirements, features, acceptance criteria
- @.project/tech-stack.md — Technology choices, architecture, dependencies
- @.project/build-plan.md — Current task progress and phase tracking
- @.project/changelog.md — Version history and recent changes
- @.project/issues/ISSUE_TEMPLATE.md — Issue template for bug/debt/security findings

## Environment

- **Go 1.26** installed at `/usr/local/go/bin/go`
- Agents MUST export PATH before any Go command: `export PATH=/usr/local/go/bin:$HOME/go/bin:$PATH`
- GitHub repo: `github.com/ul0gic/ctsnare` (public, SSH signing with ul0gic@corelift.io)

## Commands

```bash
# Development (run directly)
go run ./cmd/ctsnare

# Build
go build -o ctsnare ./cmd/ctsnare

# Test
go test ./...

# Test with race detection
go test -race -count=1 ./...

# Lint
golangci-lint run ./...

# Format
gofmt -w .

# Vet
go vet ./...

# Full verification (run at merge gates)
go build -o ctsnare ./cmd/ctsnare && go vet ./... && golangci-lint run ./... && go test ./...
```

## Project Structure

```
ctsnare/
|-- .project/                # Project documentation
|   |-- prd.md               # Product Requirements Document
|   |-- tech-stack.md        # Technology choices and rationale
|   |-- build-plan.md        # Orchestration manifest and task tracking
|   |-- changelog.md         # Version history
|   |-- issues/              # Issue tracking
|-- .claude/                 # Agent and rule definitions
|-- cmd/
|   |-- ctsnare/
|       |-- main.go          # Entry point — calls internal/cmd.Execute()
|-- internal/
|   |-- domain/              # Core types, interfaces, contracts (FROZEN after Phase 1)
|   |   |-- types.go         # Hit, CTLogEntry, ScoredDomain, Severity
|   |   |-- interfaces.go    # Scorer, Store, ProfileLoader interfaces
|   |   |-- query.go         # QueryFilter, DBStats
|   |   |-- profile.go       # Profile type
|   |-- cmd/                 # Cobra subcommand definitions
|   |   |-- root.go          # Root command, persistent flags
|   |   |-- watch.go         # watch command (TUI + headless)
|   |   |-- query.go         # query command (CLI search)
|   |   |-- db.go            # db management commands
|   |   |-- profiles.go      # profile listing/inspection
|   |   |-- output.go        # Shared output formatting (table/json/csv)
|   |-- poller/              # CT log polling engine
|   |   |-- ctlog.go         # RFC 6962 API client
|   |   |-- parser.go        # Certificate parsing, domain extraction
|   |   |-- poller.go        # Per-log goroutine polling loop
|   |   |-- manager.go       # Multi-poller lifecycle management
|   |-- scoring/             # Domain scoring engine
|   |   |-- scorer.go        # Score calculation + severity classification
|   |   |-- heuristics.go    # Individual scoring heuristics
|   |-- profile/             # Keyword profile management
|   |   |-- profile.go       # Profile manager (ProfileLoader impl)
|   |   |-- builtin.go       # Built-in profiles (crypto, phishing, all)
|   |-- storage/             # SQLite data layer
|   |   |-- db.go            # Database init, WAL mode, connection
|   |   |-- schema.go        # Table definitions and indexes
|   |   |-- hits.go          # Hit CRUD — upsert, query
|   |   |-- sessions.go      # Session management, clear, stats
|   |   |-- export.go        # JSONL and CSV export
|   |-- tui/                 # Bubbletea TUI
|   |   |-- app.go           # Root model — view switching, key routing
|   |   |-- feed.go          # Live feed view
|   |   |-- explorer.go      # DB explorer view
|   |   |-- detail.go        # Record drill-down view
|   |   |-- filter.go        # Filter input overlay
|   |   |-- styles.go        # Lipgloss style definitions
|   |   |-- keys.go          # Key binding definitions
|   |-- config/              # Configuration loading
|       |-- config.go        # TOML parsing, defaults, flag merging
|-- go.mod
|-- go.sum
|-- .golangci.yml            # Linter configuration
|-- Makefile                 # Build targets (created in Phase 4)
```

## Coding Standards

Standards are auto-loaded from `.claude/rules/`. Universal rules always apply. Path-scoped rules activate only when touching matching files.

**Universal (always active):**

- `context-management.md` — Planning, context window discipline, verification loops
- `build-discipline.md` — Zero tolerance for warnings/errors, commit discipline
- `code-quality.md` — DRY, clear over clever, error handling, file organization
- `testing.md` — Test behavior not implementation, error paths, edge cases
- `security.md` — No hardcoded secrets, input validation, HTTPS, security headers
- `self-improvement.md` — Keep CLAUDE.md current, recognize skill/rule opportunities
- `orchestration.md` — File ownership, worktree discipline, merge gates, agent coordination

**Path-scoped (activate on matching files):**

- `typescript.md` — Strict mode, no `any`, Zod at boundaries, React rules
- `rust.md` — No `unsafe` without justification, clippy zero warnings, error handling
- `go.md` — Always check errors, golangci-lint clean, context on external calls
- `swift.md` — No force unwrap, SwiftLint zero warnings, HIG compliance
- `python.md` — Type hints everywhere, ruff zero warnings, pytest, async discipline
- `css.md` — Design tokens, no `!important`, WCAG AA, fluid typography
- `api-design.md` — REST conventions, consistent errors, pagination, no N+1

## Available Agents

Use these by switching to the appropriate agent when the task matches.

| Agent | Use When |
|-------|----------|
| `backend-engineer` | Go/Rust backend work, API design, database |
| `frontend-engineer` | UI/UX, animations, creative frontend |
| `code-review-engineer` | Full codebase or PR review |
| `security-engineer` | Security audits, penetration testing |
| `refactor-engineer` | Restructuring, boundary enforcement |
| `lint-engineer` | Multi-language lint remediation, AST codemods |
| `extension-engineer` | Browser extensions, userscripts, page augmentation |
| `aws-architect` | AWS infrastructure, serverless, CDK/SAM/Terraform, `aws` CLI |
| `azure-architect` | Azure infrastructure, App Service/Functions, Bicep/Terraform, `az` CLI |
| `cloudflare-architect` | Cloudflare Workers, D1, R2, KV, Durable Objects, Wrangler, Terraform |
| `macos-engineer` | macOS/Swift desktop apps |
| `ios-engineer` | iOS/Swift mobile development |
| `api-intelligence-analyst` | API traffic analysis, data forensics |
| `build-plan-architect` | Create orchestration-aware build plans from PRD + tech-stack |
| `cli-engineer` | CLI tools, TUI apps, Bubble Tea, Ratatui, Textual, Ink |
| `devops-engineer` | CI/CD, Docker, GitHub Actions, deployment, monitoring |
| `documentation-engineer` | API docs, architecture docs, READMEs, changelogs, runbooks |
| `qa-engineer` | Testing strategy, test automation, coverage, E2E |

## Available Skills

| Skill | What It Does |
|-------|-------------|
| `/review` | Run structured code review with severity ratings |
| `/commit` | Stage, commit with good message, push, open PR |
| `/security-check` | Security audit against OWASP/CWE standards |
| `/refactor` | Analyze codebase and produce phased refactor plan |
| `/lint-fix [file]` | Fix all lint errors in a specific file |
| `/plan-project` | Generate orchestration-aware build plan with parallelization, merge gates, agent assignments |

## Issue Management

When you discover bugs, security issues, performance problems, or technical debt during any work:
1. Create an issue file in `.project/issues/open/` using the `ISSUE_TEMPLATE.md` format
2. Name it `ISSUE-XXX-short-description.md` (increment from highest existing number)
3. Fill in severity, type, affected files, and suggested fix
4. Continue your current work — issues are tracked, not blockers unless CRITICAL

## Parallel Phase Protocol (Agent Teams)

For parallel build plan phases, use proper agent teams — NOT bare Task subagents:

1. `TeamCreate` to set up a named team
2. `TaskCreate` to populate the shared task list from build-plan.md tasks
3. Spawn agents with `team_name` + `isolation: "worktree"` for both communication AND file isolation
4. Agents use `SendMessage` for coordination, `TaskList`/`TaskUpdate` for shared progress tracking
5. Lead monitors via automatic message delivery — do NOT poll agents
6. Merge gate after all teammates finish

**Do NOT** launch plain `Task` subagents with worktree isolation for parallel phases — they can't communicate or see each other's progress.

## Critical Rules

- Always read relevant project docs before making changes
- Run build/test commands after every task — zero warnings, zero errors
- Never commit secrets, .env files, or credentials
- Update `.project/build-plan.md` after completing tasks
- Update `.project/changelog.md` at milestones
- File issues for bugs/debt discovered during work — don't silently ignore problems
- Respect file ownership boundaries during parallel phases
