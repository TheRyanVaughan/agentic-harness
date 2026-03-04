package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// Cursor implements the Backend interface for Cursor CLI.
type Cursor struct{}

func NewCursor() *Cursor {
	return &Cursor{}
}

func (c *Cursor) Name() string { return "cursor" }

func (c *Cursor) Run(ctx context.Context, opts RunOpts) (*RunResult, error) {
	prompt, err := os.ReadFile(opts.PromptFile)
	if err != nil {
		return nil, fmt.Errorf("reading prompt file: %w", err)
	}

	// Cursor has no --max-budget-usd, so we enforce timeout as the budget guard.
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	args := []string{
		"-p", string(prompt),
		"--yolo",
		"--output-format", "json",
	}
	if opts.Model != "" {
		args = append(args, "--model", opts.Model)
	}
	if opts.WorkDir != "" {
		args = append(args, "--workspace", opts.WorkDir)
	}

	start := time.Now()
	cmd := exec.CommandContext(ctx, "cursor", args...)
	if opts.WorkDir != "" {
		cmd.Dir = opts.WorkDir
	}

	output, err := cmd.Output()
	duration := time.Since(start)

	result := &RunResult{
		Duration: duration,
	}
	if cmd.ProcessState != nil {
		result.ExitCode = cmd.ProcessState.ExitCode()
	}

	if opts.OutputFile != "" {
		if writeErr := os.WriteFile(opts.OutputFile, output, 0o644); writeErr != nil {
			return result, fmt.Errorf("writing output file: %w", writeErr)
		}
		result.OutputFile = opts.OutputFile
	}

	result.CostUSD, result.TokensUsed = parseCursorOutput(output)

	if err != nil {
		return result, fmt.Errorf("cursor exited with error: %w", err)
	}
	return result, nil
}

func (c *Cursor) ReviewRun(ctx context.Context, opts ReviewOpts) (*RunResult, error) {
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

	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	args := []string{
		"-p", prompt,
		"--mode", "ask",
		"--output-format", "json",
	}
	if opts.Model != "" {
		args = append(args, "--model", opts.Model)
	}
	if opts.WorkDir != "" {
		args = append(args, "--workspace", opts.WorkDir)
	}

	start := time.Now()
	cmd := exec.CommandContext(ctx, "cursor", args...)
	if opts.WorkDir != "" {
		cmd.Dir = opts.WorkDir
	}

	output, err := cmd.Output()
	duration := time.Since(start)

	result := &RunResult{
		Duration: duration,
	}
	if cmd.ProcessState != nil {
		result.ExitCode = cmd.ProcessState.ExitCode()
	}
	result.CostUSD, result.TokensUsed = parseCursorOutput(output)

	if err != nil {
		return result, fmt.Errorf("cursor review exited with error: %w", err)
	}
	return result, nil
}

func parseCursorOutput(output []byte) (costUSD float64, tokensUsed int) {
	var parsed struct {
		CostUSD    float64 `json:"cost_usd"`
		TokensUsed int     `json:"total_tokens"`
	}
	if err := json.Unmarshal(output, &parsed); err != nil {
		return 0, 0
	}
	return parsed.CostUSD, parsed.TokensUsed
}
