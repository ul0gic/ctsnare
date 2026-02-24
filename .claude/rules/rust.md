---
paths:
  - "**/*.rs"
  - "**/Cargo.toml"
---

# Rust Rules

## Ownership & Safety

- No `unsafe` blocks without explicit justification and a safety comment
- Prefer borrowing over cloning — clone only when necessary
- Use lifetimes explicitly when the compiler needs help, not just to silence errors
- Prefer `&str` over `String` for function parameters when ownership isn't needed
- Use `Cow<str>` when you might or might not need ownership

## Error Handling

- Use `thiserror` for library errors, `anyhow` for application errors
- No `.unwrap()` in production code — use `?`, `.expect("reason")`, or handle the error
- No `panic!` in library code
- Propagate errors with `?` — add context with `.context("what failed")`
- Custom error types for domain-specific failures

## Clippy — Zero Warnings

- `#![warn(clippy::all, clippy::pedantic)]` at crate level
- No `#[allow]` attributes without justification
- Address every clippy suggestion or document why it's suppressed

## Performance

- Minimize allocations in hot paths
- Use iterators over indexed loops
- Prefer `Vec::with_capacity` when size is known
- Use `#[inline]` sparingly and only with benchmarks to prove benefit
- Profile before optimizing — use `cargo flamegraph` or `perf`

## Concurrency

- Prefer `tokio` for async runtime
- Use `Arc<Mutex<T>>` only when shared mutable state is truly needed
- Prefer channels (mpsc, broadcast) over shared state
- Use `Send + Sync` bounds explicitly
- No busy-waiting — use proper async primitives

## Structure

- One type per module when the type is complex
- `mod.rs` or module file — pick a convention and stick with it
- `pub` only what needs to be public — default to private
- Group related types in the same module
- Tests in the same file (`#[cfg(test)]` module) for unit tests
