---
name: lint-fix
description: Fix all lint errors in a specific file or set of files methodically.
disable-model-invocation: true
argument-hint: "[file-path]"
---

# Lint Fix

Use the knowledge from the `lint-engineer` agent instructions when fixing lint errors.

## Target

Fix lint errors in: $ARGUMENTS

## Process

1. **Detect language** — determine the language and linter from file extension and project config:
   - TypeScript/JavaScript: ESLint, Biome, or oxlint
   - Python: ruff, mypy, pyright
   - Go: golangci-lint
   - Rust: clippy
   - CSS: stylelint
2. **Run lint** — execute the appropriate linter on the target file(s)
3. **Read the file completely** — understand context before fixing
4. **Group errors by type** — identify patterns, count frequencies
5. **Assess automation** — if 5+ instances of the same mechanical pattern, consider codemod
6. **Fix all errors** — work methodically through each type:
   - Use proper type narrowing and validation at boundaries
   - Use language-idiomatic patterns (type guards in TS, error wrapping in Go, `?` in Rust)
   - Remove unused imports and variables
   - Fix async/promise/concurrency patterns
7. **Verify zero errors** — run linter again, must show zero errors and zero warnings
8. **Verify build** — ensure changes don't break the build

## Rules

- Never disable lint rules with comments (eslint-disable, # noqa, #[allow(...)], //nolint)
- Never use type-system escape hatches (@ts-ignore, type: ignore, unsafe without justification)
- Fix the actual code to comply with rules — never suppress
- If a fix requires a type that doesn't exist yet, use `unknown` with narrowing or file an issue
