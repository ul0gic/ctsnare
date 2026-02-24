---
paths:
  - "**/*.py"
  - "**/pyproject.toml"
  - "**/requirements*.txt"
---

# Python Rules

## Type Safety

- Type hints on all function signatures — parameters and return types
- Use `mypy` or `pyright` in strict mode
- No `Any` unless interfacing with untyped libraries — narrow it immediately
- Use `TypedDict`, `dataclass`, or Pydantic models for structured data
- Use `Protocol` for structural subtyping over ABC when possible

## Error Handling

- No bare `except:` — always catch specific exceptions
- No `except Exception:` without re-raising or explicit justification
- No silent `pass` in except blocks — log or handle it
- Use custom exception classes for domain-specific errors
- Context managers (`with`) for all resource management

## Style

- Follow PEP 8 — enforced by ruff or black, not manually
- `ruff` for linting and formatting — zero warnings
- No `# noqa` without justification
- f-strings over `.format()` or `%` formatting
- Use `pathlib.Path` over `os.path`
- Use `enum.Enum` for fixed sets of values

## Structure

- One class per file for non-trivial classes
- `__init__.py` should be minimal — no business logic
- Use `src/` layout for packages
- Separate CLI entry points from library code
- Dependency injection through constructor parameters

## Async

- Use `asyncio` with `async/await` — no callback-based patterns
- No mixing sync and async — pick one per boundary
- Use `asyncio.gather` for concurrent tasks
- Proper cancellation handling with `asyncio.CancelledError`
- Use `httpx` over `requests` for async HTTP

## Testing

- `pytest` as the test framework — no unittest
- Use `pytest` fixtures over setup/teardown methods
- Use `pytest.raises` for exception testing
- Parametrize tests for multiple input cases
- Use `unittest.mock.patch` sparingly — prefer dependency injection

## Dependencies

- Use `pyproject.toml` for project configuration
- Pin dependencies in lock files
- Virtual environments always — never install globally
- Prefer `uv` or `poetry` for dependency management
