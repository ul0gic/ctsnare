---
paths:
  - "**/*.ts"
  - "**/*.tsx"
  - "**/*.js"
  - "**/*.jsx"
---

# TypeScript Rules

## Strict Mode — No Exceptions

- `strict: true` in tsconfig — always
- No `any` — use `unknown` and narrow with type guards
- No `@ts-ignore` — fix the type issue
- No `@ts-expect-error` — fix the type issue
- No `as` type assertions unless absolutely proven necessary — prefer type guards
- No `// eslint-disable` comments — never disable lint rules, fix the code

## Type Safety

- Validate at boundaries — API responses, user input, external data
- Use Zod or equivalent for runtime validation at system edges
- Derive types from schemas with `z.infer<typeof schema>` — don't duplicate types manually
- Exhaustive switch statements — no `default` on enums you control
- Use discriminated unions to make invalid states unrepresentable
- Use `as const` for literal types
- Use `satisfies` over `as` for type checking without widening

## Async

- Always await or handle promises — no floating promises
- Use `void` keyword for intentionally ignored promises in event handlers
- Add `.catch()` or try/catch — no unhandled rejections
- Use AbortController for cancellable operations
- Timeouts on all external calls

## React (when applicable)

- Exhaustive useEffect dependencies — trust the linter
- Clean up effects — return cleanup functions
- No inline object/array creation in JSX props (causes re-renders)
- Custom hooks for reusable stateful logic
- Prefer composition over prop drilling

## Imports

- No circular imports
- No barrel file abuse that creates circular dependency nightmares
- Explicit imports — no `import *`

## Naming

- Interfaces: PascalCase, no `I` prefix
- Types: PascalCase
- Enums: PascalCase members
- Constants: UPPER_SNAKE_CASE for true constants
- Functions/variables: camelCase
- Files: match the primary export name
