# Code Style Rules

<!-- These rules are provided to both the implementer and reviewer agents.
     Add rules as you discover them in code review. Each rule should be
     specific and verifiable. "Write better tests" is bad. "Use [Theory]+
     [InlineData] instead of multiple [Fact] methods" is good. -->


## Testing Conventions

### Philosophy
- Tests document BEHAVIOR, not implementation
- Test the interface, not the internals
- Every test should fail for exactly one reason

### Naming
<!-- Fill in your org's test naming convention. Examples: -->
- C#: `MethodName_WhenCondition_ShouldExpectedResult`
- Go: `TestMethodName_WhenCondition_ShouldExpectedResult` with t.Run() subtests
- TypeScript: describe("MethodName") > it("should X when Y")

### Parameterization Rules
- C#: Use `[Theory]` + `[InlineData]` or `[MemberData]` — NOT multiple `[Fact]` methods
- Go: Use table-driven tests with `t.Run()` subtests
- TypeScript: Use `describe` blocks with `test.each()` — NOT separate `it()` blocks
- If >3 inputs test the same behavior, it MUST be parameterized

### Gate Functions (check BEFORE writing each test)
1. "Is this a distinct behavior, or a variation of an already-tested behavior?"
   → If variation: add as a row in an existing parameterized test
   → If distinct: write a new test method
2. "Am I testing real component behavior or just that a mock exists?"
   → If testing mock existence: delete the assertion or remove the mock
3. "Is my mock setup >50% of the test code?"
   → If yes: you're testing mocks, not behavior. Restructure.
4. Count test methods per file. If >8: refactor into parameterized tests.

### Anti-Patterns (NEVER do these)
- Separate test methods that differ only in input values
- Copy-pasted setup across tests — extract a helper
- Testing implementation details instead of behavior
- "I'll mock this to be safe" — mock only what you understand
- Tests that pass for wrong reasons


## File Structure Rules

<!-- FILL IN: Your org's directory conventions. Examples: -->
<!-- - New endpoints go in: [path] -->
<!-- - Shared helpers live in: [path] -->
<!-- - Test files live alongside source files / in a separate __tests__ dir -->
<!-- - Never create a new top-level directory without approval -->


## Code Conventions

<!-- FILL IN: Your org's patterns. Examples: -->
<!-- - Error handling: [pattern] -->
<!-- - Logging: [library and pattern] -->
<!-- - Dependency injection: [pattern] -->
<!-- - API response format: [pattern] -->


## PR Review Severity

When reviewing, tag each issue:
- `[MUST FIX]` — Blocks merge. Bugs, missing requirements, convention violations.
- `[SHOULD FIX]` — Strong recommendation. Tech debt, missing abstractions.
- `[NIT]` — Style preference. Non-blocking.

Review priority order: Correctness > Security > Structure > Tests > Maintainability > Style


## Learned Rules

<!-- Add one rule here every time a PR review catches a mistake.
     Date it so you know when patterns emerged.
     Format: YYYY-MM-DD: [specific, verifiable rule] -->
