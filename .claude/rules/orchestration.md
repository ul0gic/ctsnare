# Orchestration

## File Ownership

- Every file belongs to one agent at a time — check the build plan's file ownership map
- Never modify files outside your assigned boundary during parallel phases
- If you need a change in another agent's territory, note it as a dependency for the merge gate
- Shared files (package.json, lock files, tsconfig, lint config) are OFF LIMITS during parallel phases
- Only the lead session modifies `.project/` files (build-plan.md, changelog.md)

## Worktree Discipline

- During parallel phases, each agent works in their own worktree
- Commit frequently to your branch — small, atomic commits
- Run your agent-specific build verification in your worktree, not the full project build
- Never switch to another agent's worktree
- When your phase work is complete, commit everything and signal done

## Merge Gates

- All parallel work STOPS at merge gates — no agent continues past a gate until merge is clean
- Merge gate protocol:
  1. All agents commit and push their branches
  2. Lead session merges branches into main one at a time
  3. Full build/lint/test suite runs on merged result
  4. Conflicts resolved before any agent proceeds
  5. All agents pull fresh main before next phase
- If merge causes lint/build failures, the agent whose changes broke it fixes it

## Agent Teams

- When running as a team, claim tasks from the shared task list before starting
- Mark tasks in-progress immediately — prevents another teammate from duplicating work
- Mark tasks complete only after build verification passes
- Communicate blockers via team messaging — don't silently wait
- Stay in your lane — the task list defines your work, not your initiative

## Cloud Offload

- Tasks marked ☁️ in the build plan are eligible for cloud offload
- Cloud tasks must be fully independent — no interactive feedback needed
- Cloud tasks must have clear completion criteria the lead can verify
- Check cloud task results with `/tasks` before depending on their output
- Pull cloud results back with `/teleport` when needed

## Build Verification During Parallel Phases

- Each agent runs verification scoped to their directory only:
  - Backend: `pnpm --filter backend typecheck && pnpm --filter backend lint`
  - Frontend: `pnpm --filter frontend typecheck && pnpm --filter frontend lint`
  - Shared: `pnpm --filter shared typecheck`
- Full project verification (`pnpm ci:check` or equivalent) runs ONLY at merge gates
- Never run full project lint in a worktree during parallel phases — it will see incomplete work from other agents and fail

## Preventing Collisions

- Read the build plan's conflict zones table before starting any phase
- If a task requires touching a conflict zone file, it MUST be in a sequential phase
- Package dependency additions: collect all needed dependencies, add them in ONE commit at the merge gate
- Config file changes: same — collect, merge, apply once
- If two agents need the same file modified, the build plan should have sequenced this — if it didn't, stop and flag it

## Signaling

- When an agent completes their parallel phase work: commit, push, and update the build plan task status
- When blocked: immediately flag with ⛔ status and describe the blocker
- When a merge gate is reached: all agents stop and wait for lead to merge
- When resuming after a merge gate: pull fresh main, verify your worktree is clean, then proceed
