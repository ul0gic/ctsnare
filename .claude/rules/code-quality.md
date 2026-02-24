# Code Quality

## Principles

- Clear over clever — readable code wins, every time
- DRY — don't repeat yourself, but never at the expense of clarity or boundaries
- Duplication is better than the wrong abstraction
- Delete dead code — don't comment it out, don't keep it "just in case"
- No premature abstraction — wait until you have 3 real use cases
- No over-engineering — solve the current problem, not hypothetical future ones

## Functions

- Small, focused, single responsibility
- Meaningful names that describe what they do
- Explicit return types on exported/public functions
- No side effects in functions that look pure
- If a function is hard to name, it's doing too much

## Error Handling

- No silent failures — handle every error explicitly
- Propagate context with errors — include what happened and where
- Fail fast — detect problems early, provide clear messages
- Use typed errors, not string errors
- Never swallow exceptions with empty catch blocks

## Comments

- Comments explain *why*, never *what* — the code shows what
- No commented-out code — that's what git is for
- No TODO comments without a linked issue or task
- No obvious comments (`// increment counter` above `counter++`)

## File Organization

- One primary export per file
- Group imports: external packages, then internal, then relative
- Keep files under 300 lines — split if larger
- Consistent file naming within the project

## Dependencies

- Minimize dependency count — fewer deps = fewer problems
- Audit before adding — is this really needed or can you write it?
- Keep dependencies updated
- Lock file always committed
