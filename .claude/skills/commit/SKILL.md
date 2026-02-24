---
name: commit
description: Stage changes, write a good commit message, push, and optionally open a PR.
disable-model-invocation: true
---

# Commit, Push & PR

## Process

1. **Check status** — run `git status` and `git diff` to understand all changes
2. **Stage files** — add relevant files by name (never `git add -A` blindly, exclude .env/secrets)
3. **Write commit message** — summarize the *why* not the *what*, follow conventional commit style if the project uses it
4. **Commit** — create the commit
5. **Push** — push to remote with `-u` flag if needed
6. **PR (if requested)** — if $ARGUMENTS includes "pr", create a pull request with:
   - Short title (under 70 chars)
   - Summary section with bullet points
   - Test plan section

## Rules

- Never force push
- Never commit .env, credentials, or secrets
- Always review the diff before committing
- If there are no changes, say so and stop
