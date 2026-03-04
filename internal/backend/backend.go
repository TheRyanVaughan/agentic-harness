package backend

import (
	"context"
	"time"
)

// Backend abstracts over different agent CLIs (Claude Code, Cursor).
type Backend interface {
	// Name returns the backend identifier ("claude" or "cursor").
	Name() string

	// Run executes the agent with the given prompt in the given workdir.
	Run(ctx context.Context, opts RunOpts) (*RunResult, error)

	// ReviewRun executes the agent in read-only mode for code review.
	ReviewRun(ctx context.Context, opts ReviewOpts) (*RunResult, error)
}

// RunOpts configures an agent run.
type RunOpts struct {
	PromptFile string
	WorkDir    string
	Model      string
	BudgetUSD  float64
	MaxTurns   int
	Timeout    time.Duration
	OutputFile string // path to write JSON output
}

// ReviewOpts configures a review agent run.
type ReviewOpts struct {
	DiffFile  string
	StyleFile string // path to STYLE.md, may be empty
	WorkDir   string
	Model     string
	BudgetUSD float64
	Timeout   time.Duration
}

// RunResult holds the outcome of an agent run.
type RunResult struct {
	ExitCode   int
	OutputFile string
	CostUSD    float64
	TokensUsed int
	Duration   time.Duration
}
