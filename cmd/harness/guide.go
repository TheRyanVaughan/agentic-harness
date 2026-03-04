package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newGuideCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "guide",
		Short: "Step-by-step guide to using the harness",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Print(guideText)
		},
	}
}

const guideText = `
HARNESS — GETTING STARTED GUIDE
================================

This guide walks you through every step, in order, to go from zero to a
completed agent-driven task.


PREREQUISITES
─────────────
  • Go 1.23+ installed (to build harness)
  • Claude Code CLI ("claude") or Cursor CLI ("cursor") installed and authenticated
  • "just" command runner installed (https://github.com/casey/just)
  • A git repository to work in


STEP 1: BUILD & INSTALL HARNESS
────────────────────────────────
Run this once (or after pulling updates):

  just build          # builds ./harness binary
  # or
  just install        # installs to $GOPATH/bin/harness


STEP 2: INITIALIZE YOUR REPO
─────────────────────────────
cd into your project's git repo, then run:

  harness init

This creates:
  • AGENTS.md    — instructions the agent follows during execution
  • STYLE.md     — code style rules (edit this to match your project)
  • justfile     — verification recipes with "verify", "lint", "test" targets
  • .gitignore   — adds .agent/ so state files aren't committed

DO THIS NOW: Open the generated justfile and wire up your project's real
lint, typecheck, and test commands. The "just verify" recipe is what harness
runs to check the agent's work.


STEP 3: (OPTIONAL) CONFIGURE DEFAULTS
──────────────────────────────────────
Create a harness.toml in your repo root (or ~/.config/harness/config.toml
for global defaults). See config.example.toml for all options. Example:

  [defaults]
  backend    = "claude"
  model      = "opus"
  budget_usd = 5.00
  max_retries = 3
  timeout    = "10m"
  verify_cmd = "just verify"

  [review]
  model      = "sonnet"
  budget_usd = 2.00

You can also set HARNESS_BACKEND, HARNESS_MODEL, etc. as env vars.
CLI flags always take highest priority.


STEP 4: WRITE A SPEC
─────────────────────
Create a spec file describing what you want the agent to build:

  harness new-spec my-feature

This creates spec-my-feature.md from a template. Open it and fill in:
  • Context     — relevant files, current behavior, background
  • Requirements — what must be done (checkboxes)
  • Constraints  — what must NOT happen
  • Acceptance   — testable assertions
  • Escalation   — when the agent should stop and ask

The better your spec, the better the result. Be specific.


STEP 5: RUN THE STATE MACHINE
──────────────────────────────
Start the agent:

  harness run --spec spec-my-feature.md

What happens automatically:
  1. EXECUTING  — the agent writes code based on your spec
  2. VERIFYING  — harness runs "just verify" to check the work
  3. RETRYING   — if verify fails, the error is fed back and the agent tries again
                  (up to max_retries times)
  4. REVIEWING  — a review agent checks the diff for code quality
  5. FIXING     — if the review found issues, the agent fixes them
  6. HUMAN_REVIEW — harness exits (code 2) and waits for you

You can watch progress in another terminal:

  harness status

Useful flags:
  --backend claude|cursor    Override backend
  --model opus               Override model
  --budget 10.0              Set max spend in USD
  --max-retries 5            More retry attempts
  --timeout 15m              Longer timeout
  --dry-run                  Print the assembled prompt without executing


STEP 6: REVIEW THE AGENT'S WORK
────────────────────────────────
When harness exits with code 2, it's waiting for your review. Look at:

  • The code changes: git diff
  • The state file:   cat .agent/state.json
  • The review output: cat .agent/review.md (if present)

Test the changes yourself — run the app, check the UI, verify edge cases.


STEP 7: APPROVE OR REJECT
──────────────────────────
If you're happy with the changes:

  harness approve       # marks task complete, exits 0

If the changes need more work:

  harness reject        # returns to SPEC_READY state

After rejecting, edit your spec to clarify what went wrong, then run
harness again (Step 5). The agent will try again with the updated spec.


STEP 8: HANDLE "STUCK" STATE
─────────────────────────────
If the agent exceeds max retries, harness exits with code 3 (STUCK).
This means automated attempts couldn't pass verification. You should:

  1. Read the error output in .agent/
  2. Fix the issue manually, OR simplify the spec
  3. Run harness again


MULTI-TASK WORKFLOW
───────────────────
For parallel tasks, use git worktrees:

  git worktree add -b feat-auth ../my-project-auth
  cd ../my-project-auth
  harness run --spec spec-auth.md

  # Back in main repo, check all tasks:
  harness status --all

Each worktree gets its own .agent/ directory and independent state.


STANDALONE CODE REVIEW
──────────────────────
Review any uncommitted changes without running the full state machine:

  harness review


QUICK REFERENCE
───────────────
  harness init                   Scaffold repo with templates
  harness new-spec <name>        Create a spec file
  harness run --spec <file>      Run the state machine
  harness status [--all] [--json]  Check task status
  harness approve                Accept changes
  harness reject                 Reject changes, revise spec
  harness review                 Standalone code review
  harness guide                  Show this guide

EXIT CODES
──────────
  0 = Complete (task done)
  1 = Error (harness itself failed)
  2 = Human Review (needs your verification)
  3 = Stuck (max retries exceeded, manual intervention needed)
`
