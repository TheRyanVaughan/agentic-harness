# agentic-harness

A deterministic state machine CLI for agent-driven development. Wraps Claude Code and Cursor CLIs with retry loops, automated code review, budget guards, and human checkpoints.

## Install

```bash
go install github.com/ryanvaughan/agentic-harness/cmd/harness@latest
```

Or build from source:

```bash
go build -o harness ./cmd/harness
```

## Quick Start

```bash
# 1. Initialize a repo with templates
harness init

# 2. Create a spec
harness new-spec my-feature

# 3. Edit spec-my-feature.md with requirements

# 4. Run the state machine
harness run --spec spec-my-feature.md

# 5. Check status
harness status

# 6. After reviewing changes
harness approve    # or: harness reject
```

## Commands

| Command | Description |
|---------|-------------|
| `harness run --spec FILE` | Run the full state machine |
| `harness status [--all] [--json]` | Show task status |
| `harness approve` | Accept changes, mark complete |
| `harness reject` | Reject changes, return to spec revision |
| `harness review` | Standalone code review on current diff |
| `harness init` | Scaffold AGENTS.md, STYLE.md, justfile |
| `harness new-spec NAME` | Create spec from template |

## State Machine

```
SPEC READY ──► EXECUTING ──► VERIFYING ──► REVIEWING ──► HUMAN REVIEW ──► COMPLETE
   ▲               ▲            │              │              │
   │               │            ▼              ▼              │
   │            RETRYING     STUCK          FIXING            │
   │                                                          │
   └──────────────────────── REJECT ──────────────────────────┘
```

**Human states** (harness exits, waits for you):
- `SPEC READY` — Write/revise the spec
- `HUMAN REVIEW` — Verify changes in deployed env
- `STUCK` — Max retries exceeded, intervene manually

**Autonomous states** (harness + agent, no human needed):
- `EXECUTING`, `VERIFYING`, `RETRYING`, `REVIEWING`, `FIXING`

## Exit Codes

| Code | State | Meaning |
|------|-------|---------|
| 0 | Complete | Task done |
| 1 | Error | Harness itself failed |
| 2 | Human Review | Needs human verification |
| 3 | Stuck | Max retries exceeded |

## Configuration

Config is loaded from (highest priority first):
1. CLI flags
2. Environment variables (`HARNESS_BACKEND`, `HARNESS_MODEL`, etc.)
3. `./harness.toml` (per-repo)
4. `~/.config/harness/config.toml` (global)

See `config.example.toml` for all options.

## Backends

### Claude Code
- Budget enforced via `--max-budget-usd`
- Turn limit via `--max-turns`
- Review uses `--allowedTools "Read,Grep,Glob"` for read-only mode

### Cursor
- No budget flag — timeout used as fallback constraint
- Review uses `--mode ask` for read-only mode

## `.agent/` Directory

Created per run, should be gitignored:

```
.agent/
├── state.json      # Current state (updated on every transition)
├── prompt.md       # Assembled prompt
├── run-N.json      # Agent output per attempt
├── diff.patch      # Git diff at review time
├── review.md       # Review agent output
└── progress.md     # Agent's progress log
```
