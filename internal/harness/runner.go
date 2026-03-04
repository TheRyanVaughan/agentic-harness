package harness

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/ryanvaughan/agentic-harness/internal/backend"
	"github.com/ryanvaughan/agentic-harness/internal/config"
	"github.com/ryanvaughan/agentic-harness/internal/review"
)

// ExitCode constants for deterministic scripting.
const (
	ExitComplete    = 0
	ExitError       = 1
	ExitHumanReview = 2
	ExitStuck       = 3
)

// Result holds the outcome of a harness run.
type Result struct {
	State    State
	Cost     float64
	Attempts int
	Duration time.Duration
}

// ExitCode returns the exit code for the result state.
func (r *Result) ExitCode() int {
	switch r.State {
	case StateComplete:
		return ExitComplete
	case StateHumanReview:
		return ExitHumanReview
	case StateStuck:
		return ExitStuck
	default:
		return ExitError
	}
}

// Runner executes the state machine loop.
type Runner struct {
	backend   backend.Backend
	reviewer  *review.Reviewer
	config    *config.Config
	stateFile *StateFile
	state     State
	workDir   string
	agentDir  string
	prompt    *Prompt
}

// RunnerOpts holds options for creating a runner.
type RunnerOpts struct {
	Backend    backend.Backend
	Config     *config.Config
	WorkDir    string
	SpecFile   string
	TaskName   string
	AgentsFile string
	StyleFile  string
}

// NewRunner creates a new runner.
func NewRunner(opts RunnerOpts) (*Runner, error) {
	agentDir := filepath.Join(opts.WorkDir, ".agent")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating .agent dir: %w", err)
	}

	sf := NewStateFile(agentDir)
	sf.Task = opts.TaskName
	sf.Spec = opts.SpecFile
	sf.Backend = opts.Backend.Name()
	sf.Model = opts.Config.Model
	sf.MaxRetries = opts.Config.MaxRetries

	styleFile := opts.StyleFile
	if styleFile == "" {
		candidate := filepath.Join(opts.WorkDir, "STYLE.md")
		if _, err := os.Stat(candidate); err == nil {
			styleFile = candidate
		}
	}

	reviewer := review.NewReviewer(
		opts.Backend,
		opts.Config.Review.Model,
		opts.Config.Review.BudgetUSD,
		styleFile,
	)

	r := &Runner{
		backend:   opts.Backend,
		reviewer:  reviewer,
		config:    opts.Config,
		stateFile: sf,
		state:     StateSpecReady,
		workDir:   opts.WorkDir,
		agentDir:  agentDir,
	}

	// Assemble prompt
	agentsFile := opts.AgentsFile
	if agentsFile == "" {
		candidate := filepath.Join(opts.WorkDir, "AGENTS.md")
		if _, err := os.Stat(candidate); err == nil {
			agentsFile = candidate
		}
	}

	prompt, err := NewPrompt(agentDir, opts.SpecFile, agentsFile, styleFile)
	if err != nil {
		return nil, fmt.Errorf("assembling prompt: %w", err)
	}
	r.prompt = prompt

	return r, nil
}

// Run executes the full state machine loop.
func (r *Runner) Run(ctx context.Context) (*Result, error) {
	attempt := 0
	var totalCost float64
	start := time.Now()

	r.transition(StateExecuting)

	for {
		select {
		case <-ctx.Done():
			return &Result{
				State:    r.state,
				Cost:     totalCost,
				Attempts: attempt,
				Duration: time.Since(start),
			}, ctx.Err()
		default:
		}

		switch r.state {
		case StateExecuting:
			attempt++
			outputFile := filepath.Join(r.agentDir, fmt.Sprintf("run-%d.json", attempt))

			runCtx, cancel := context.WithTimeout(ctx, r.config.Timeout)
			result, err := r.backend.Run(runCtx, backend.RunOpts{
				PromptFile: r.prompt.Path(),
				WorkDir:    r.workDir,
				Model:      r.config.Model,
				BudgetUSD:  r.config.BudgetUSD,
				MaxTurns:   r.config.MaxTurns,
				Timeout:    r.config.Timeout,
				OutputFile: outputFile,
			})
			cancel()

			if result != nil {
				totalCost += result.CostUSD
			}
			r.stateFile.CostUSD = totalCost
			r.stateFile.Attempt = attempt

			if err != nil {
				// Agent error is not necessarily fatal — move to verify
				fmt.Fprintf(os.Stderr, "warning: agent run error: %v\n", err)
			}
			r.transition(StateVerifying)

		case StateVerifying:
			verifyOut, verifyErr := r.runVerify(ctx)
			if verifyErr == nil {
				r.transition(StateReviewing)
			} else if attempt < r.config.MaxRetries {
				if err := r.prompt.AppendRetryContext(verifyErr, verifyOut); err != nil {
					return nil, fmt.Errorf("appending retry context: %w", err)
				}
				r.transition(StateRetrying)
			} else {
				r.transition(StateStuck)
			}

		case StateRetrying:
			r.transition(StateExecuting)

		case StateReviewing:
			diffFile, err := r.generateDiff()
			if err != nil {
				// If we can't generate a diff, skip review
				fmt.Fprintf(os.Stderr, "warning: could not generate diff: %v\n", err)
				r.transition(StateHumanReview)
				continue
			}

			reviewResult, err := r.reviewer.Run(ctx, diffFile, r.workDir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: review failed: %v\n", err)
				r.transition(StateHumanReview)
				continue
			}

			totalCost += reviewResult.CostUSD
			r.stateFile.CostUSD = totalCost

			// Write review output
			_ = os.WriteFile(filepath.Join(r.agentDir, "review.md"), []byte(reviewResult.RawOutput), 0o644)

			if reviewResult.Clean {
				r.transition(StateHumanReview)
			} else {
				if err := r.prompt.AppendReviewFeedback(reviewResult.Issues); err != nil {
					return nil, fmt.Errorf("appending review feedback: %w", err)
				}
				r.transition(StateFixing)
			}

		case StateFixing:
			attempt++
			outputFile := filepath.Join(r.agentDir, fmt.Sprintf("run-%d.json", attempt))

			runCtx, cancel := context.WithTimeout(ctx, r.config.Timeout)
			result, err := r.backend.Run(runCtx, backend.RunOpts{
				PromptFile: r.prompt.Path(),
				WorkDir:    r.workDir,
				Model:      r.config.Model,
				BudgetUSD:  r.config.BudgetUSD,
				MaxTurns:   r.config.MaxTurns,
				Timeout:    r.config.Timeout,
				OutputFile: outputFile,
			})
			cancel()

			if result != nil {
				totalCost += result.CostUSD
			}
			r.stateFile.CostUSD = totalCost
			r.stateFile.Attempt = attempt

			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: fix run error: %v\n", err)
			}
			r.transition(StateVerifying)

		case StateHumanReview:
			r.printSummary(totalCost, attempt, time.Since(start))
			return &Result{
				State:    StateHumanReview,
				Cost:     totalCost,
				Attempts: attempt,
				Duration: time.Since(start),
			}, nil

		case StateStuck:
			r.printStuckDiagnostic(attempt, totalCost)
			return &Result{
				State:    StateStuck,
				Cost:     totalCost,
				Attempts: attempt,
				Duration: time.Since(start),
			}, nil

		case StateComplete:
			return &Result{
				State:    StateComplete,
				Cost:     totalCost,
				Attempts: attempt,
				Duration: time.Since(start),
			}, nil
		}
	}
}

// Resume resumes from a persisted state (e.g., after `harness approve`).
func (r *Runner) Resume(ctx context.Context, newState State) (*Result, error) {
	r.transition(newState)
	if newState == StateComplete {
		return &Result{
			State:    StateComplete,
			Cost:     r.stateFile.CostUSD,
			Attempts: r.stateFile.Attempt,
		}, nil
	}
	return r.Run(ctx)
}

func (r *Runner) transition(to State) {
	r.state = to
	r.stateFile.State = to
	r.stateFile.NeedsHuman = to.NeedsHuman()
	r.stateFile.Action = StateInfos[to].Action
	r.stateFile.UpdatedAt = time.Now()
	r.stateFile.History = append(r.stateFile.History, HistoryEntry{
		State: to,
		At:    time.Now(),
	})
	if err := r.stateFile.Persist(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to persist state: %v\n", err)
	}
}

func (r *Runner) runVerify(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "sh", "-c", r.config.VerifyCmd)
	cmd.Dir = r.workDir

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	return out.String(), err
}

func (r *Runner) generateDiff() (string, error) {
	cmd := exec.Command("git", "diff", "HEAD")
	cmd.Dir = r.workDir

	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	diffFile := filepath.Join(r.agentDir, "diff.patch")
	if err := os.WriteFile(diffFile, out, 0o644); err != nil {
		return "", err
	}
	return diffFile, nil
}

func (r *Runner) printSummary(cost float64, attempts int, duration time.Duration) {
	fmt.Printf("\n=== Harness: Human Review Needed ===\n")
	fmt.Printf("  Backend:  %s\n", r.backend.Name())
	fmt.Printf("  Model:    %s\n", r.config.Model)
	fmt.Printf("  Attempts: %d\n", attempts)
	fmt.Printf("  Cost:     $%.2f\n", cost)
	fmt.Printf("  Duration: %s\n", duration.Round(time.Second))
	fmt.Printf("\nVerify changes, then run:\n")
	fmt.Printf("  harness approve   # accept and mark complete\n")
	fmt.Printf("  harness reject    # revise spec and re-run\n")
}

func (r *Runner) printStuckDiagnostic(attempts int, cost float64) {
	fmt.Printf("\n=== Harness: Stuck (max retries exceeded) ===\n")
	fmt.Printf("  Attempts: %d / %d\n", attempts, r.config.MaxRetries)
	fmt.Printf("  Cost:     $%.2f\n", cost)
	fmt.Printf("\nCheck .agent/ for logs:\n")
	fmt.Printf("  .agent/run-*.json   # agent outputs\n")
	fmt.Printf("  .agent/prompt.md    # assembled prompt with retry context\n")
	fmt.Printf("\nOptions:\n")
	fmt.Printf("  1. Fix manually and run `harness approve`\n")
	fmt.Printf("  2. Revise spec and run `harness run --spec spec.md`\n")
}
