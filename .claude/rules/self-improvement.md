# Self-Improvement

## Keep CLAUDE.md Current

When any of these change, update CLAUDE.md immediately:

- New dev/build/test/lint commands added or changed
- Project structure changes (new directories, renamed modules)
- New agents or skills added to `.claude/`
- New dependencies that affect how the project runs
- New environment variables or configuration requirements

Do not wait to be asked. If you changed it, update the index.

## Keep .project/ Docs Current

- Update `build-plan.md` task status after completing work
- Update `changelog.md` at milestones or significant changes
- Update `tech-stack.md` when adding new dependencies or tools
- If `prd.md` requirements change based on user decisions, note the change

## Recognize Skill Opportunities

Create a new skill when you notice:

- The same multi-step workflow requested 3+ times
- A complex sequence that would benefit from a single `/command`
- A task that should always follow the same structured process

When creating a skill:
- Place it in `.claude/skills/<name>/SKILL.md`
- Use `context: fork` for analysis tasks that don't need to modify files
- Use `agent:` frontmatter to pair with the right agent when applicable
- Include clear instructions so the skill works without user hand-holding
- Update CLAUDE.md's Available Skills table

## Recognize Rule Opportunities

Suggest a new rule when you notice:

- The same correction or pattern enforced repeatedly
- A project convention that should apply automatically
- A mistake that keeps happening and could be prevented

Do not create rules silently — propose them and let the user decide.
