---
name: plan-project
description: Generate an orchestration-aware build plan from PRD and tech-stack
disable-model-invocation: true
argument-hint: "[optional: specific focus areas or constraints]"
---

# /plan-project

Delegate this task to the `build-plan-architect` agent.

## Prerequisites Check

Before creating the plan, verify these files exist and are complete:
1. `.project/prd.md` — Must have requirements, features, and acceptance criteria defined
2. `.project/tech-stack.md` — Must have technology choices and architecture documented

If either file is missing or incomplete, stop and tell the user what's needed.

## Process

1. **Read all inputs:**
   - `.project/prd.md`
   - `.project/tech-stack.md`
   - `.claude/CLAUDE.md`
   - Every file in `.claude/agents/`

2. **Analyze the project scope:**
   - What are the domain boundaries? (backend, frontend, shared, infra, data, etc.)
   - Which agents map to which boundaries?
   - What are the natural parallelization opportunities?
   - What are the conflict zones? (shared files, config, package manifests)

3. **Build the orchestration plan:**
   - Design phases with proper dependency chains
   - Assign agents with file ownership boundaries
   - Insert merge gates between parallel windows
   - Define execution strategy per phase
   - Map parallelization visually
   - Identify cloud-offloadable tasks

4. **Write the build plan:**
   - Output the complete plan to `.project/build-plan.md`
   - Include every section defined in the build-plan-architect agent's output format
   - Every task must be specific enough to execute without clarification

5. **Present summary to user:**
   - Total phases and tasks
   - Maximum parallelism windows
   - Agent assignments
   - Estimated parallel vs sequential split
   - Key merge gates
   - Conflict zones to watch

## User Arguments

If the user provides arguments with `/plan-project`, incorporate them:
- Specific constraints ("no cloud offloading", "single engineer only")
- Focus areas ("prioritize backend first", "MVP only")
- Agent preferences ("use agent teams instead of subagents")

$ARGUMENTS
