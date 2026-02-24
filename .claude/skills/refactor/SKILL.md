---
name: refactor
description: Analyze codebase structure and produce a phased refactor plan with concrete tasks.
disable-model-invocation: true
---

# Refactor Analysis

Delegate this task to the `refactor-engineer` agent. This is a **read-only analysis** — do not modify any files.

## Target

Analyze: $ARGUMENTS

If no target specified, analyze the entire codebase.

## Process

1. **Map what exists vs what is used** — find dead code, unused exports, phantom abstractions
2. **Identify real domains** — products, pages, features, modules
3. **Evaluate boundaries** — are they real or folder decoration? Cross-imports?
4. **Assess naming** — files, folders, functions, components — do they describe what they are?
5. **Trace dependencies** — import graph, circular deps, coupling hotspots
6. **Identify false abstractions** — shared utils that shouldn't be shared, premature generalization

## Output

Produce a phased refactor plan:

### Phase 1: Safe Moves (no behavior change)
- File/folder renames and relocations
- Dead code deletion
- Import cleanup

### Phase 2: Boundary Enforcement
- Extract/isolate domains
- Break circular dependencies
- Establish clear ownership

### Phase 3: Structural Improvement
- Consolidate truly shared code
- Simplify state/data flow
- Optimize hot paths

Each phase broken into concrete, reviewable tasks with file paths and descriptions.
