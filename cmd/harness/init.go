package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ryanvaughan/agentic-harness/templates"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Scaffold a repo with AGENTS.md, STYLE.md, justfile, and .gitignore",
		RunE: func(cmd *cobra.Command, args []string) error {
			wd, _ := os.Getwd()

			files := []struct {
				name     string
				template string
			}{
				{"AGENTS.md", "AGENTS.md"},
				{"STYLE.md", "STYLE.md"},
				{"justfile", "justfile"},
			}

			for _, f := range files {
				dest := filepath.Join(wd, f.name)
				if !force {
					if _, err := os.Stat(dest); err == nil {
						fmt.Printf("  skip %s (exists, use --force to overwrite)\n", f.name)
						continue
					}
				}

				data, err := templates.FS.ReadFile(f.template)
				if err != nil {
					return fmt.Errorf("reading template %s: %w", f.template, err)
				}

				if err := os.WriteFile(dest, data, 0o644); err != nil {
					return fmt.Errorf("writing %s: %w", f.name, err)
				}
				fmt.Printf("  created %s\n", f.name)
			}

			// Append .agent/ to .gitignore
			gitignore := filepath.Join(wd, ".gitignore")
			if err := appendToGitignore(gitignore, ".agent/"); err != nil {
				return fmt.Errorf("updating .gitignore: %w", err)
			}
			fmt.Println("  updated .gitignore")

			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing files")
	return cmd
}

func newNewSpecCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "new-spec [name]",
		Short: "Create a new spec file from template",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := "task"
			if len(args) > 0 {
				name = args[0]
			}

			dest := fmt.Sprintf("spec-%s.md", name)
			if _, err := os.Stat(dest); err == nil {
				return fmt.Errorf("%s already exists", dest)
			}

			data, err := templates.FS.ReadFile("spec.template.md")
			if err != nil {
				return fmt.Errorf("reading spec template: %w", err)
			}

			if err := os.WriteFile(dest, data, 0o644); err != nil {
				return fmt.Errorf("writing %s: %w", dest, err)
			}

			fmt.Printf("Created %s\n", dest)
			return nil
		},
	}
}

func appendToGitignore(path, entry string) error {
	existing, _ := os.ReadFile(path)
	if strings.Contains(string(existing), entry) {
		return nil
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	prefix := ""
	if len(existing) > 0 && existing[len(existing)-1] != '\n' {
		prefix = "\n"
	}
	_, err = f.WriteString(prefix + entry + "\n")
	return err
}
