package backend

import (
	"fmt"
	"os"
	"strings"
)

// buildReviewPrompt constructs the two-stage review prompt from a diff,
// optional style rules, and optional spec file.
func buildReviewPrompt(diff, styleFile, specFile string) string {
	var b strings.Builder

	b.WriteString(`You are a code reviewer. Perform two review passes on this diff, then give a final verdict.

## PASS 1 — SPEC COMPLIANCE
`)

	if specFile != "" {
		specContent, err := os.ReadFile(specFile)
		if err == nil && len(specContent) > 0 {
			b.WriteString(fmt.Sprintf(`Read the task spec below, then compare it to the actual diff.
Do NOT trust what the implementer claimed — verify by reading the code.

Check:
- Is everything in the spec actually implemented (not stubbed or skipped)?
- Is anything implemented that was NOT in the spec (scope creep)?
- Do the acceptance criteria from the spec all appear to be satisfied?

<spec>
%s
</spec>

`, string(specContent)))
		}
	} else {
		b.WriteString("No spec file available. Skip this pass.\n\n")
	}

	b.WriteString(`## PASS 2 — CODE QUALITY
`)

	if styleFile != "" {
		styleContent, err := os.ReadFile(styleFile)
		if err == nil && len(styleContent) > 0 {
			b.WriteString(fmt.Sprintf(`Use these style rules when reviewing:

<style-rules>
%s
</style-rules>

`, string(styleContent)))
		}
	}

	b.WriteString(`Review the diff in this priority order:
1. CORRECTNESS: Logic errors, edge cases, error handling, null safety
2. SECURITY: Injection, auth, data exposure, input validation
3. STRUCTURE: Files in right directories, following existing patterns, right abstractions
4. TESTS: Parameterized, behavior-focused, using helpers, no test bloat
5. MAINTAINABILITY: Naming, separation of concerns, duplication
6. STYLE: Consistency with codebase

Tag each issue with severity:
- [MUST FIX] — Blocks merge. Bugs, missing requirements, convention violations, security.
- [SHOULD FIX] — Strong recommendation. Tech debt, missing abstractions, poor patterns.
- [NIT] — Style/naming preference. Non-blocking.

## VERDICT

If there are NO [MUST FIX] issues, respond with exactly 'LGTM' on the first line,
followed by any [SHOULD FIX] or [NIT] items.

If there ARE [MUST FIX] issues, list all issues grouped by file. Do NOT say LGTM.

`)

	b.WriteString(fmt.Sprintf("```diff\n%s\n```\n", diff))

	return b.String()
}
