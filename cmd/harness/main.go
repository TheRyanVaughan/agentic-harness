package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "harness",
		Short: "Agentic workflow harness — deterministic state machine for agent-driven development",
	}

	root.AddCommand(
		newRunCmd(),
		newStatusCmd(),
		newApproveCmd(),
		newRejectCmd(),
		newInitCmd(),
		newNewSpecCmd(),
		newReviewCmd(),
		newGuideCmd(),
	)

	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
