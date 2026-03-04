# Agent Instructions

## Workflow
1. Read the spec file provided in your prompt
2. Derive a plan and present it before writing code
3. After implementing, run `just verify` and fix any failures
4. Maintain .agent/progress.md — check off tasks, log decisions and blockers

## On Failure
If verification fails, read the full error output, diagnose the root cause,
fix the issue, and re-run `just verify`. Do NOT report done if any check is failing.

## Constraints
- Do not modify files outside the scope defined in the spec
- Do not push to remote
- Do not install new dependencies without noting it in progress.md
- If you are unsure about an architectural decision, note it in progress.md
  and proceed with your best judgment — do not stop and ask
