# Agent Instructions

## Workflow
1. Read the spec file provided in your prompt
2. Derive a plan and present it before writing code
3. After implementing, run `just verify` and fix any failures
4. Maintain .agent/progress.md — check off tasks, log decisions and blockers

## Critical Rules
0. NEVER create new directories without explicit approval in the spec
1. NEVER modify files outside the scope defined in the spec
2. NEVER write separate test methods for input variations — use parameterized tests
3. NEVER duplicate logic — extract shared code into helpers
4. ALWAYS follow the pattern of the reference service/component named in the spec
5. ALWAYS stage files individually when committing (never `git add .`)
6. ALWAYS read STYLE.md before writing any code

## Deviation Rules

**Rule 1 — Auto-fix bugs:** Fix broken logic, type errors, null refs, validation
gaps inline. Add/update tests for the fix.

**Rule 2 — Auto-add critical missing pieces:** Error handling, input validation,
null checks, auth guards, logging. Required for correctness.

**Rule 3 — Auto-fix blockers:** Missing deps, broken imports, wrong types,
build errors.

**Rule 4 — STOP for architecture:** New database tables, schema changes, new
service layers, framework changes, breaking API changes. Note in progress.md
and stop.

Priority: Rule 4 (stop) > Rules 1-3 (auto-fix) > uncertainty (treat as Rule 4).
Max 3 auto-fix attempts per issue, then document in progress.md and move on.
Only fix issues caused by the current task. Log pre-existing problems without fixing.

## Git Commits
- Commit after each completed task, not at the end of all work
- Stage files individually (never `git add .` or `git add -A`)
- Format: `{type}({scope}): {description}`
- Types: feat, fix, test, refactor, docs, chore
- Each commit must be independently revertable
- Do not push to remote

## On Failure
If verification fails, read the full error output, diagnose the root cause,
fix the issue, and re-run `just verify`. Do NOT report done if any check is failing.

## Constraints
- Do not install new dependencies without noting it in progress.md
- If you are unsure about a Rule 4 decision, note it in progress.md and stop
