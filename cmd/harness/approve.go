package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ryanvaughan/agentic-harness/internal/harness"
	"github.com/spf13/cobra"
)

func newApproveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "approve",
		Short: "Approve changes and mark task complete",
		RunE: func(cmd *cobra.Command, args []string) error {
			return transitionFromHumanReview(harness.StateComplete)
		},
	}
}

func newRejectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reject",
		Short: "Reject changes and return to spec revision",
		RunE: func(cmd *cobra.Command, args []string) error {
			return transitionFromHumanReview(harness.StateSpecReady)
		},
	}
}

func transitionFromHumanReview(to harness.State) error {
	wd, _ := os.Getwd()
	stateFile := filepath.Join(wd, ".agent", "state.json")

	sf, err := harness.LoadStateFile(stateFile)
	if err != nil {
		return fmt.Errorf("no active task found (.agent/state.json missing)")
	}

	if sf.State != harness.StateHumanReview && sf.State != harness.StateStuck {
		return fmt.Errorf("task is in state %q, not human_review or stuck", sf.State.String())
	}

	sf.State = to
	sf.NeedsHuman = to.NeedsHuman()
	sf.Action = harness.StateInfos[to].Action
	sf.UpdatedAt = time.Now()
	sf.History = append(sf.History, harness.HistoryEntry{
		State: to,
		At:    time.Now(),
	})

	if err := sf.Persist(); err != nil {
		return fmt.Errorf("persisting state: %w", err)
	}

	switch to {
	case harness.StateComplete:
		fmt.Printf("Task %q marked complete.\n", sf.Task)
		fmt.Printf("  Total cost: $%.2f\n", sf.CostUSD)
		fmt.Printf("  Attempts:   %d\n", sf.Attempt)
	case harness.StateSpecReady:
		fmt.Printf("Task %q rejected. Revise spec and re-run:\n", sf.Task)
		fmt.Printf("  harness run --spec %s\n", sf.Spec)
	}

	return nil
}
