---
paths:
  - "**/*.go"
  - "**/go.mod"
---

# Go Rules

## Error Handling

- Always check errors — no `_` for error returns unless explicitly justified
- Wrap errors with context: `fmt.Errorf("doing X: %w", err)`
- Return errors, don't panic — panic is for truly unrecoverable situations
- Use sentinel errors or custom error types for errors callers need to inspect
- Errors are values — treat them that way

## Linting — Zero Warnings

- `golangci-lint run` must pass clean
- No `//nolint` directives without justification
- Enable: `errcheck`, `govet`, `staticcheck`, `unused`, `gosimple`, `ineffassign`

## Concurrency

- No goroutine leaks — always ensure goroutines can exit
- Use `context.Context` for cancellation and timeouts on every external call
- Prefer channels for communication, mutexes for state protection
- Use `sync.WaitGroup` or `errgroup` to wait for goroutines
- Never use `time.Sleep` for synchronization

## Style

- Accept interfaces, return structs
- Keep interfaces small — 1-3 methods
- No interface pollution — don't create interfaces until you need abstraction
- Use table-driven tests
- Package names: short, lowercase, no underscores

## Structure

- `internal/` for private packages
- One package per directory
- No circular imports between packages
- `cmd/` for entry points, domain packages for business logic
- Dependency injection through constructor functions

## Performance

- Preallocate slices when size is known: `make([]T, 0, n)`
- Reuse buffers with `sync.Pool` for high-throughput paths
- Use `strings.Builder` for string concatenation
- Profile with `pprof` before optimizing
- Connection pooling for database and HTTP clients
