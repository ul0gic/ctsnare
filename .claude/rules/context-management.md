# Context Management

## Planning

- Start complex tasks in plan mode — think before coding
- Read relevant files before editing — understand what exists
- Read CLAUDE.md and .project/ docs before making architectural decisions
- Break large tasks into smaller steps — one behavior change at a time

## Context Window

- Don't try to hold the entire codebase in context
- Use subagents for exploration and research — keep main context clean
- Compact proactively at ~50% utilization, don't wait for auto-compact
- When context is getting full, summarize findings before continuing

## Verification Loop

- Always verify your own work — run build, tests, lint after changes
- If you can't verify, say so — don't claim something works without checking
- Use the feedback loop: make change, verify, fix, verify again
- Never say "should work" — prove it works
