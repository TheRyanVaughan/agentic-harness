package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/ryanvaughan/agentic-harness/internal/harness"
	"github.com/spf13/cobra"
)

type statusEntry struct {
	Task       string  `json:"task"`
	Worktree   string  `json:"worktree"`
	Spec       string  `json:"spec"`
	State      string  `json:"state"`
	NeedsHuman bool    `json:"needs_human"`
	Action     string  `json:"action"`
	Attempts   int     `json:"attempts"`
	CostUSD    float64 `json:"cost_usd"`
	Duration   string  `json:"duration,omitempty"`
	UpdatedAt  string  `json:"updated_at"`
}

func newStatusCmd() *cobra.Command {
	var (
		all     bool
		jsonOut bool
	)

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show task status",
		RunE: func(cmd *cobra.Command, args []string) error {
			var entries []statusEntry

			if all {
				entries = scanWorktrees()
			} else {
				entry, err := currentStatus()
				if err != nil {
					return err
				}
				if entry != nil {
					entries = append(entries, *entry)
				}
			}

			if len(entries) == 0 {
				fmt.Println("No active tasks found.")
				return nil
			}

			if jsonOut {
				data, _ := json.MarshalIndent(entries, "", "  ")
				fmt.Println(string(data))
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
			fmt.Fprintf(w, "TASK\tSTATE\tACTOR\tACTION NEEDED\n")
			fmt.Fprintf(w, "────\t─────\t─────\t─────────────\n")
			for _, e := range entries {
				actor := "AUTO"
				if e.NeedsHuman {
					actor = "HUMAN"
				}
				if e.State == "complete" {
					actor = "—"
				}
				action := e.Action
				if action == "" {
					action = "—"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", e.Task, titleCase(e.State), actor, action)
			}
			w.Flush()
			return nil
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "Show all tracked tasks across worktrees")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Machine-readable JSON output")

	return cmd
}

func currentStatus() (*statusEntry, error) {
	wd, _ := os.Getwd()
	return currentStatusFromDir(wd)
}

func scanWorktrees() []statusEntry {
	var entries []statusEntry
	wd, _ := os.Getwd()

	if entry, err := currentStatusFromDir(wd); err == nil {
		entries = append(entries, *entry)
	}

	gitDir := filepath.Join(wd, ".git")
	if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
		worktreesDir := filepath.Join(gitDir, "worktrees")
		wtEntries, _ := os.ReadDir(worktreesDir)
		for _, wt := range wtEntries {
			if !wt.IsDir() {
				continue
			}
			gitdirFile := filepath.Join(worktreesDir, wt.Name(), "gitdir")
			data, err := os.ReadFile(gitdirFile)
			if err != nil {
				continue
			}
			wtPath := filepath.Dir(strings.TrimSpace(string(data)))
			if entry, err := currentStatusFromDir(wtPath); err == nil {
				entries = append(entries, *entry)
			}
		}
	}

	return entries
}

func currentStatusFromDir(dir string) (*statusEntry, error) {
	stateFile := filepath.Join(dir, ".agent", "state.json")
	sf, err := harness.LoadStateFile(stateFile)
	if err != nil {
		return nil, err
	}
	return &statusEntry{
		Task:       sf.Task,
		Worktree:   dir,
		Spec:       sf.Spec,
		State:      sf.State.String(),
		NeedsHuman: sf.NeedsHuman,
		Action:     sf.Action,
		Attempts:   sf.Attempt,
		CostUSD:    sf.CostUSD,
		UpdatedAt:  sf.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}, nil
}

func titleCase(s string) string {
	words := strings.Split(strings.ReplaceAll(s, "_", " "), " ")
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}
