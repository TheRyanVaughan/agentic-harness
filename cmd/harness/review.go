package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"

	"github.com/ryanvaughan/agentic-harness/internal/config"
	"github.com/ryanvaughan/agentic-harness/internal/review"
	"github.com/spf13/cobra"
)

func newReviewCmd() *cobra.Command {
	var (
		backendStr string
		model      string
	)

	cmd := &cobra.Command{
		Use:   "review",
		Short: "Run standalone code review on current diff",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			if cmd.Flags().Changed("backend") {
				cfg.Backend = backendStr
			}
			if cmd.Flags().Changed("model") {
				cfg.Review.Model = model
			}

			b, err := selectBackend(cfg.Backend)
			if err != nil {
				return err
			}

			wd, _ := os.Getwd()

			// Generate diff
			diffOut, err := exec.Command("git", "-C", wd, "diff", "HEAD").Output()
			if err != nil {
				return fmt.Errorf("generating diff: %w", err)
			}
			if len(diffOut) == 0 {
				fmt.Println("No changes to review.")
				return nil
			}

			diffFile := filepath.Join(wd, ".agent", "diff.patch")
			if err := os.MkdirAll(filepath.Dir(diffFile), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(diffFile, diffOut, 0o644); err != nil {
				return err
			}

			styleFile := filepath.Join(wd, "STYLE.md")
			if _, err := os.Stat(styleFile); err != nil {
				styleFile = ""
			}

			reviewer := review.NewReviewer(b, cfg.Review.Model, cfg.Review.BudgetUSD, styleFile)

			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
			defer cancel()

			result, err := reviewer.Run(ctx, diffFile, wd)
			if err != nil {
				return fmt.Errorf("review failed: %w", err)
			}

			if result.Clean {
				fmt.Println("LGTM — no issues found.")
			} else {
				fmt.Println("Review found issues:")
				fmt.Println(result.Issues)
			}

			fmt.Printf("\nCost: $%.2f\n", result.CostUSD)
			return nil
		},
	}

	cmd.Flags().StringVar(&backendStr, "backend", "", "Backend: claude or cursor")
	cmd.Flags().StringVar(&model, "model", "", "Model for review")

	return cmd
}
