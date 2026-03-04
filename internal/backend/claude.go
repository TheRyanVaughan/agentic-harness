package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// Claude implements the Backend interface for Claude Code CLI.
type Claude struct{}

func NewClaude() *Claude {
	return &Claude{}
}

func (c *Claude) Name() string { return "claude" }

func (c *Claude) Run(ctx context.Context, opts RunOpts) (*RunResult, error) {
	prompt, err := os.ReadFile(opts.PromptFile)
	if err != nil {
		return nil, fmt.Errorf("reading prompt file: %w", err)
	}

	args := []string{
		"-p", string(prompt),
		"--dangerously-skip-permissions",
		"--output-format", "json",
	}
	if opts.Model != "" {
		args = append(args, "--model", opts.Model)
	}
	if opts.BudgetUSD > 0 {
		args = append(args, "--max-budget-usd", fmt.Sprintf("%.2f", opts.BudgetUSD))
	}
	if opts.MaxTurns > 0 {
		args = append(args, "--max-turns", fmt.Sprintf("%d", opts.MaxTurns))
	}

	start := time.Now()
	cmd := exec.CommandContext(ctx, "claude", args...)
	cmd.Dir = opts.WorkDir

	output, err := cmd.Output()
	duration := time.Since(start)

	result := &RunResult{
		Duration: duration,
	}

	if cmd.ProcessState != nil {
		result.ExitCode = cmd.ProcessState.ExitCode()
	}

	// Write output to file
	if opts.OutputFile != "" {
		if writeErr := os.WriteFile(opts.OutputFile, output, 0o644); writeErr != nil {
			return result, fmt.Errorf("writing output file: %w", writeErr)
		}
		result.OutputFile = opts.OutputFile
	}

	// Parse cost/tokens from JSON output
	result.CostUSD, result.TokensUsed = parseClaudeOutput(output)

	if err != nil {
		return result, fmt.Errorf("claude exited with error: %w", err)
	}
	return result, nil
}

func (c *Claude) ReviewRun(ctx context.Context, opts ReviewOpts) (*RunResult, error) {
	diff, err := os.ReadFile(opts.DiffFile)
	if err != nil {
		return nil, fmt.Errorf("reading diff file: %w", err)
	}

	prompt := fmt.Sprintf("Review this diff for code quality issues. If everything looks good, respond with exactly 'LGTM'. Otherwise, list specific issues.\n\n```diff\n%s\n```", string(diff))

	if opts.StyleFile != "" {
		styleContent, err := os.ReadFile(opts.StyleFile)
		if err == nil && len(styleContent) > 0 {
			prompt = fmt.Sprintf("Use these style rules when reviewing:\n\n%s\n\n%s", string(styleContent), prompt)
		}
	}

	args := []string{
		"-p", prompt,
		"--allowedTools", "Read,Grep,Glob",
		"--output-format", "json",
	}
	if opts.Model != "" {
		args = append(args, "--model", opts.Model)
	}
	if opts.BudgetUSD > 0 {
		args = append(args, "--max-budget-usd", fmt.Sprintf("%.2f", opts.BudgetUSD))
	}

	start := time.Now()
	cmd := exec.CommandContext(ctx, "claude", args...)
	cmd.Dir = opts.WorkDir

	output, err := cmd.Output()
	duration := time.Since(start)

	result := &RunResult{
		Duration: duration,
	}
	if cmd.ProcessState != nil {
		result.ExitCode = cmd.ProcessState.ExitCode()
	}
	result.CostUSD, result.TokensUsed = parseClaudeOutput(output)

	if err != nil {
		return result, fmt.Errorf("claude review exited with error: %w", err)
	}
	return result, nil
}

// parseClaudeOutput extracts cost and token usage from Claude JSON output.
func parseClaudeOutput(output []byte) (costUSD float64, tokensUsed int) {
	var parsed struct {
		CostUSD    float64 `json:"cost_usd"`
		TokensUsed int     `json:"total_tokens"`
		Usage      struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(output, &parsed); err != nil {
		return 0, 0
	}
	costUSD = parsed.CostUSD
	tokensUsed = parsed.TokensUsed
	if tokensUsed == 0 {
		tokensUsed = parsed.Usage.InputTokens + parsed.Usage.OutputTokens
	}
	return
}
