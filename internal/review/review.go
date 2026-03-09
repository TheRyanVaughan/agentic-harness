package review

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/ryanvaughan/agentic-harness/internal/backend"
)

// Result holds the outcome of a code review.
type Result struct {
	Clean      bool    // true if LGTM, no issues found
	Issues     string  // issue descriptions (empty if clean)
	RawOutput  string  // raw agent output
	CostUSD    float64
	TokensUsed int
}

// Reviewer runs code review via an agent backend.
type Reviewer struct {
	backend   backend.Backend
	model     string
	budgetUSD float64
	styleFile string
	specFile  string
}

// NewReviewer creates a new Reviewer.
func NewReviewer(b backend.Backend, model string, budgetUSD float64, styleFile, specFile string) *Reviewer {
	return &Reviewer{
		backend:   b,
		model:     model,
		budgetUSD: budgetUSD,
		styleFile: styleFile,
		specFile:  specFile,
	}
}

// Run performs a two-stage code review (spec compliance + code quality) on the given diff file.
func (r *Reviewer) Run(ctx context.Context, diffFile, workDir string) (*Result, error) {
	runResult, err := r.backend.ReviewRun(ctx, backend.ReviewOpts{
		DiffFile:  diffFile,
		StyleFile: r.styleFile,
		SpecFile:  r.specFile,
		WorkDir:   workDir,
		Model:     r.model,
		BudgetUSD: r.budgetUSD,
	})
	if err != nil {
		return nil, fmt.Errorf("review run failed: %w", err)
	}

	// Read the output to determine if LGTM or has issues
	rawOutput := ""
	if runResult.OutputFile != "" {
		data, err := os.ReadFile(runResult.OutputFile)
		if err == nil {
			rawOutput = extractResultText(data)
		}
	}

	result := &Result{
		RawOutput:  rawOutput,
		CostUSD:    runResult.CostUSD,
		TokensUsed: runResult.TokensUsed,
	}

	// Check if the review is clean (LGTM)
	upper := strings.ToUpper(rawOutput)
	if strings.Contains(upper, "LGTM") && !strings.Contains(upper, "NOT LGTM") {
		result.Clean = true
	} else {
		result.Issues = rawOutput
	}

	return result, nil
}

// extractResultText pulls the result text from Claude's JSON output.
func extractResultText(data []byte) string {
	var output struct {
		Result string `json:"result"`
	}
	if err := json.Unmarshal(data, &output); err != nil {
		return string(data)
	}
	if output.Result != "" {
		return output.Result
	}
	return string(data)
}
