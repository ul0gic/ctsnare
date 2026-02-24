---
name: review
description: Run a structured code review with severity ratings and actionable fixes.
disable-model-invocation: true
---

# Code Review

Delegate this task to the `code-review-engineer` agent. This is a **read-only analysis** — do not modify any files.

## Target

Review: $ARGUMENTS

If no target specified, review all staged changes (`git diff --cached`) or recent changes (`git diff`).

## Process

1. **Map scope** — identify all files in scope
2. **Read every file** — do not skip or summarize from names
3. **Trace imports and dependencies** — build the real dependency graph
4. **Audit security** — as if trying to break it
5. **Evaluate types** — catalog violations
6. **Check performance** — profile paths from entry to output
7. **Evaluate tests** — coverage, quality, edge cases
8. **Check configs** — tsconfig, eslint, prettier, bundler, CI, env, docker

## Output

Produce the full structured report as defined in the code-review-engineer agent, including:
- Executive summary with letter grade
- Findings by category (Security, Architecture, Type Safety, Performance, Testing, Dependencies, Dead Code)
- Scorecard table
- Priority action items (top 10)
