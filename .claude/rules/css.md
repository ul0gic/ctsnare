---
paths:
  - "**/*.css"
  - "**/*.scss"
  - "**/*.sass"
  - "**/*.less"
---

# CSS Rules

## Architecture

- Design tokens first — colors, spacing, typography defined once
- Use CSS custom properties for theming and runtime values
- No inline styles — all styling in CSS files or design system
- No `!important` — fix the specificity issue instead
- Mobile-first responsive design — min-width breakpoints, not max-width

## Layout

- Flexbox for one-dimensional, Grid for two-dimensional
- Use `gap` over margins between items
- Use `clamp()` for fluid sizing — font sizes, widths, spacing
- Container queries over media queries when sizing depends on parent
- No magic numbers — use tokens or calc

## Naming & Organization

- Consistent naming convention (BEM, utility, or modules — pick one)
- Scope styles to components — no global style leakage
- Group properties logically: layout, box model, typography, visual, animation
- One component per CSS file when using modules

## Performance

- Prefer CSS animations over JavaScript — `transform` and `opacity` are GPU-accelerated
- Use `will-change` sparingly and only when measured
- Avoid layout thrashing — batch reads and writes
- No complex selectors (deep nesting, universal selectors in hot paths)

## Accessibility

- WCAG AA contrast ratios minimum — 4.5:1 for text, 3:1 for large text
- Never use color alone to convey meaning
- Focus styles visible and clear — never `outline: none` without replacement
- Respect `prefers-reduced-motion` — disable non-essential animations
- Respect `prefers-color-scheme` — support dark and light modes

## Typography

- Use a type scale — don't invent sizes
- Fluid typography with `clamp()` — no fixed px for body text
- `rem` for font sizes, `em` for component-relative spacing
- `line-height` unitless — `1.5` not `24px`
- `text-wrap: balance` for headings
