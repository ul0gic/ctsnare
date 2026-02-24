# Project Templates

A repeatable project scaffold for structured software development with Claude Code.

## Quick Start

```bash
# Copy both .project and .claude to your new repo root
cp -r /path/to/project-templates/.project /your/project/root/
cp -r /path/to/project-templates/.claude /your/project/root/
```

Then customize:
1. Edit `.claude/CLAUDE.md` — fill in project name, description, commands, structure
2. Edit `.claude/hooks/verify.sh` — uncomment and set your build/test/lint commands
3. Remove agents you don't need from `.claude/agents/`
4. Fill out `.project/prd.md` and `tech-stack.md`
5. Run `/plan-project` to generate the orchestration-aware build plan

## Directory Structure

```
your-project/
├── .project/                        # Project documentation
│   ├── prd.md                       # Product requirements
│   ├── tech-stack.md                # Technology choices
│   ├── build-plan.md                # Orchestration manifest + task tracking
│   └── changelog.md                 # Version history
│
├── .claude/                         # Claude Code configuration
│   ├── CLAUDE.md                    # Project index (imports .project/ docs)
│   ├── settings.local.json          # Permissions, hooks config
│   ├── agents/                      # Specialist agents
│   │   ├── backend-engineer.md
│   │   ├── frontend-engineer.md
│   │   ├── build-plan-architect.md  # Orchestration-aware build planning
│   │   ├── code-review-engineer.md
│   │   ├── security-engineer.md
│   │   ├── refactor-engineer.md
│   │   ├── lint-engineer.md
│   │   ├── extension-engineer.md
│   │   ├── aws-engineer.md
│   │   ├── macos-engineer.md
│   │   ├── mobile-engineer.md
│   │   └── api-intelligence-analyst.md
│   ├── skills/                      # Slash command workflows
│   │   ├── plan-project/SKILL.md    # /plan-project — orchestration-aware build plan
│   │   ├── review/SKILL.md          # /review — structured code review
│   │   ├── commit/SKILL.md          # /commit — stage, commit, push, PR
│   │   ├── security-check/SKILL.md  # /security-check — security audit
│   │   ├── refactor/SKILL.md        # /refactor — refactor analysis & plan
│   │   └── lint-fix/SKILL.md        # /lint-fix [file] — fix lint errors
│   ├── rules/                       # Auto-loaded coding standards
│   │   ├── context-management.md    # Planning, verification loops
│   │   ├── build-discipline.md      # Zero warnings, commit discipline
│   │   ├── code-quality.md          # DRY, clear over clever
│   │   ├── testing.md               # Testing requirements
│   │   ├── security.md              # Security requirements
│   │   ├── self-improvement.md      # Keep CLAUDE.md current, suggest skills/rules
│   │   ├── orchestration.md         # File ownership, merge gates, agent coordination
│   │   ├── typescript.md            # Path-scoped: TS/JS files
│   │   ├── rust.md                  # Path-scoped: Rust files
│   │   ├── go.md                    # Path-scoped: Go files
│   │   ├── swift.md                 # Path-scoped: Swift files
│   │   ├── python.md                # Path-scoped: Python files
│   │   ├── css.md                   # Path-scoped: CSS/SCSS files
│   │   └── api-design.md            # Path-scoped: API routes/handlers
│   └── hooks/                       # Automation scripts
│       ├── format.sh                # Auto-format after every file edit
│       └── verify.sh                # Verify build/tests before stopping
│
└── [project files]
```

## Workflow

### Phase 1: Ideation (Claude Desktop)
Go back and forth on the idea in Claude Desktop. When the concept is locked, export to `prd.md`.

### Phase 2: Project Setup
1. Copy `.project/` and `.claude/` to your repo root
2. Customize `CLAUDE.md` — project name, commands, structure
3. Configure `verify.sh` — set your build/test commands
4. Trim agents you don't need

### Phase 3: Tech Stack (Claude Code)
1. Review PRD together: `Review .project/prd.md and let's decide on tech stack`
2. Fill out `.project/tech-stack.md` with technology choices and rationale

### Phase 4: Build Planning (Claude Code)
1. Run `/plan-project` — the build-plan-architect agent reads PRD + tech-stack
2. Produces orchestration-aware build plan with:
   - Phases with dependency chains
   - Agent assignments with file ownership boundaries
   - Parallelization map showing concurrent workstreams
   - Merge gates between parallel windows
   - Conflict zones identified
   - Cloud-offloadable tasks marked
   - Build verification protocol per phase
3. Review and iterate on the plan before execution

### Phase 5: Execution
Execute the build plan using the orchestration strategy it defines:
- **Sequential phases** — single agent, single working directory
- **Parallel phases (⚡)** — multiple agents in isolated worktrees
- **Cloud offload (☁️)** — fire-and-forget tasks via `&` prefix
- **Merge gates** — lead session merges branches, runs full build, resolves conflicts

### Phase 6: Ongoing
- Update task status after each completion
- Use `/commit` when ready to commit
- Update `changelog.md` at milestones
- Run `/review` before major merges

## Execution Methods

| Method | Command | Best For |
|--------|---------|----------|
| **Subagents + Worktrees** | Agent spawns with `isolation: worktree` | Parallel work within one session |
| **Agent Teams** | Enable `CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS=1` | Large parallel phases, independent workstreams |
| **Cloud Offload** | `& task description` or `claude --remote "task"` | Fire-and-forget, work while away |
| **Headless/CLI** | `claude -p "task" --allowedTools "..."` | Scripts, CI/CD, automation |

## Build Plan Features

### Status Emojis
| Icon | Status |
|------|--------|
| ⬜ | Not Started |
| 🔄 | In Progress |
| ✅ | Completed |
| ⛔ | Blocked |
| ⚠️ | Has Blockers |
| 🔍 | In Review |
| 🚫 | Skipped |
| ⏸️ | Deferred |
| ☁️ | Cloud Eligible |
| 🌳 | Worktree Required |

### Orchestration Sections
The build plan includes these orchestration-specific sections:
- **Engineer Assignments** — agents, their domain boundaries, file ownership
- **Orchestration Config** — execution strategy per phase
- **Conflict Zones** — shared files and resolution strategies
- **Build Verification Protocol** — who verifies what, when
- **Merge Gates** — sync points between parallel windows
- **Parallelization Map** — visual dependency graph

## Build Discipline

After completing each task:
1. Run build command (scoped to your directory during parallel phases)
2. Fix any warnings/errors
3. Mark task as ✅
4. Update progress summary
5. At merge gates: lead runs full project build on merged result
