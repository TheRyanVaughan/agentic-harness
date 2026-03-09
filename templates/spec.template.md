# Task: [title]

## Context
- [relevant files, current behavior, background]
- **Reference service/component:** [name of an existing similar service to follow as a pattern]

## Requirements (musts)
- [ ] [requirement 1]
- [ ] [requirement 2]

## Constraints (must-nots)
- Do not create new directories — use the existing structure
- Do not add new dependencies without noting in progress.md
- [add task-specific constraints]

## Tasks

<!-- Each task must pass this test: "Could a different Claude instance execute
     this without asking clarifying questions?" If not, make it more specific. -->

### Task 1: [action-oriented name]
- **Files:** [exact paths to create/modify]
- **Action:** [what to do and WHY — include which existing pattern to follow]
- **Verify:** [automated command, <60s, that proves this works]
- **Done:** [measurable acceptance criteria]

### Task 2: [action-oriented name]
- **Files:** [exact paths]
- **Action:** [what to do and WHY]
- **Verify:** [command]
- **Done:** [criteria]

## Acceptance Criteria
- [ ] [testable assertion an independent reviewer could verify]
- [ ] [testable assertion 2]
- [ ] All existing tests still pass
- [ ] Convention checks pass (see STYLE.md)

## Failure Modes
<!-- How could a capable agent technically complete this but produce a bad PR? -->
- [e.g., "Creates a new directory instead of using the existing XYZ directory"]
- [e.g., "Writes 10 separate test methods instead of parameterized tests"]
- [e.g., "Doesn't follow the pattern in ExistingService for this type of change"]

## Escalation Triggers
- If the task requires creating more than 3 new files, stop and ask
- If existing patterns don't clearly apply, stop and ask
- If [condition], stop and ask rather than guessing
