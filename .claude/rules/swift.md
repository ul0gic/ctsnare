---
paths:
  - "**/*.swift"
  - "**/Package.swift"
---

# Swift Rules

## The 3-Click Rule (iOS/macOS)

- Any primary user action should be reachable within 3 taps/clicks
- Navigation depth beyond 3 levels needs justification
- Critical actions (save, submit, cancel) always visible or one gesture away

## Safety — No Shortcuts

- No force unwrapping (`!`) in production code — use `guard let`, `if let`, nil coalescing
- No `try!` — handle errors or propagate with `try`
- No `as!` force casting — use `as?` with proper handling
- No implicitly unwrapped optionals unless required by framework (IBOutlets)
- Exhaustive switch statements — no `default` on enums you control

## SwiftLint — Zero Warnings

- Strict SwiftLint config, zero warnings policy
- No `// swiftlint:disable` without justification
- Follow Swift API Design Guidelines — clarity at point of use

## Type System

- Prefer structs over classes unless identity semantics are needed
- Use `let` over `var` — immutability by default
- Leverage enums with associated values for state modeling
- Make invalid states unrepresentable with the type system
- Use `@Observable` over `ObservableObject` for new code

## Concurrency

- Strict concurrency checking enabled
- `@MainActor` for all UI code — no exceptions
- No `DispatchQueue.main.async` in new code — use structured concurrency
- Use `async let` for parallel work, `TaskGroup` for dynamic parallelism
- Proper cancellation handling — check `Task.isCancelled`

## UI/UX

- Follow Apple Human Interface Guidelines — feel like a first-party app
- Dynamic Type support at every level — never hardcode font sizes
- Full VoiceOver support with meaningful accessibility labels
- Dark mode support — use semantic system colors, never hardcode
- Respect `prefers-reduced-motion` — provide animation alternatives
- Test on all supported device sizes

## Architecture

- MVVM or TCA — pick one per project and stick with it
- ViewModels own state, Views render it — no business logic in views
- Dependency injection through init — no service locators
- Protocol-based abstractions at module boundaries
- One type per file for non-trivial types

## Naming

- Follow Swift API Design Guidelines religiously
- No abbreviations — `numberOfItems` not `numItems`
- Methods read as English: `array.insert(element, at: index)`
- Bool properties: `isEnabled`, `hasContent`, `canSubmit`
