# Build Discipline

## After Every Task (ALL agents, no exceptions)

1. Run the build command — must succeed
2. Run tests — must pass
3. Run linter — zero errors, zero warnings
4. Fix anything broken before moving on
5. Update `.project/build-plan.md` — change your task status from `⬜` to `✅` immediately after completing it (use `🔄` while in progress)
6. Update `.project/changelog.md` at milestones (end of each sub-phase)
7. Steps 5-6 are MANDATORY — skipping them is a process failure

## Zero Tolerance

- Zero compiler warnings
- Zero lint errors
- Zero test failures
- No skipping verification to save time
- No "I'll fix it later" — fix it now

## Commits

- Stage specific files — never blind `git add -A`
- Review the diff before committing
- Write commit messages that explain *why*, not *what*
- Never commit broken code
- Never commit secrets, .env files, or credentials
- Never mention Claude, AI, or LLMs in commit messages — write them like a human engineer
- Never add `Co-Authored-By` lines referencing Claude or AI in commits
