package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/ryanvaughan/agentic-harness/internal/backend"
	"github.com/ryanvaughan/agentic-harness/internal/config"
	"github.com/ryanvaughan/agentic-harness/internal/harness"
	"github.com/spf13/cobra"
)

func newRunCmd() *cobra.Command {
	var (
		specFile   string
		backendStr string
		model      string
		budget     float64
		maxRetries int
		timeout    string
		verifyCmd  string
		dryRun     bool
	)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run the full state machine on a spec",
		RunE: func(cmd *cobra.Command, args []string) error {
			if specFile == "" {
				return fmt.Errorf("--spec is required")
			}
			if _, err := os.Stat(specFile); err != nil {
				return fmt.Errorf("spec file not found: %s", specFile)
			}

			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			// Apply CLI flag overrides
			if cmd.Flags().Changed("backend") {
				cfg.Backend = backendStr
			}
			if cmd.Flags().Changed("model") {
				cfg.Model = model
			}
			if cmd.Flags().Changed("budget") {
				cfg.BudgetUSD = budget
			}
			if cmd.Flags().Changed("max-retries") {
				cfg.MaxRetries = maxRetries
			}
			if cmd.Flags().Changed("timeout") {
				if d, err := time.ParseDuration(timeout); err == nil {
					cfg.Timeout = d
				}
			}
			if cmd.Flags().Changed("verify-cmd") {
				cfg.VerifyCmd = verifyCmd
			}

			b, err := selectBackend(cfg.Backend)
			if err != nil {
				return err
			}

			workDir, _ := os.Getwd()

			// Derive task name from spec filename
			taskName := filepath.Base(specFile)
			taskName = taskName[:len(taskName)-len(filepath.Ext(taskName))]

			runner, err := harness.NewRunner(harness.RunnerOpts{
				Backend:  b,
				Config:   cfg,
				WorkDir:  workDir,
				SpecFile: specFile,
				TaskName: taskName,
			})
			if err != nil {
				return fmt.Errorf("creating runner: %w", err)
			}

			if dryRun {
				promptPath := filepath.Join(workDir, ".agent", "prompt.md")
				data, err := os.ReadFile(promptPath)
				if err != nil {
					return fmt.Errorf("reading assembled prompt: %w", err)
				}
				fmt.Println("=== Assembled Prompt (dry run) ===")
				fmt.Println(string(data))
				return nil
			}

			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
			defer cancel()

			result, err := runner.Run(ctx)
			if err != nil {
				return fmt.Errorf("run failed: %w", err)
			}

			os.Exit(result.ExitCode())
			return nil
		},
	}

	cmd.Flags().StringVar(&specFile, "spec", "", "Path to the spec file (required)")
	cmd.Flags().StringVar(&backendStr, "backend", "", "Backend: claude or cursor")
	cmd.Flags().StringVar(&model, "model", "", "Model to use")
	cmd.Flags().Float64Var(&budget, "budget", 0, "Max budget in USD")
	cmd.Flags().IntVar(&maxRetries, "max-retries", 0, "Max retry attempts")
	cmd.Flags().StringVar(&timeout, "timeout", "", "Timeout duration (e.g. 10m)")
	cmd.Flags().StringVar(&verifyCmd, "verify-cmd", "", "Verification command")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print assembled prompt without executing")

	return cmd
}

func selectBackend(name string) (backend.Backend, error) {
	switch name {
	case "claude":
		return backend.NewClaude(), nil
	case "cursor":
		return backend.NewCursor(), nil
	default:
		return nil, fmt.Errorf("unknown backend: %q (use 'claude' or 'cursor')", name)
	}
}
