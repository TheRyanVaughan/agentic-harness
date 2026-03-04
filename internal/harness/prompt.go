package harness

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Prompt handles assembly and mutation of the agent prompt file.
type Prompt struct {
	dir      string // .agent/ directory
	path     string // path to the assembled prompt file
	sections []string
}

// NewPrompt creates a new prompt by assembling spec + AGENTS.md + STYLE.md.
func NewPrompt(agentDir, specFile, agentsFile, styleFile string) (*Prompt, error) {
	p := &Prompt{
		dir:  agentDir,
		path: filepath.Join(agentDir, "prompt.md"),
	}

	// Read spec
	spec, err := os.ReadFile(specFile)
	if err != nil {
		return nil, fmt.Errorf("reading spec: %w", err)
	}
	p.sections = append(p.sections, string(spec))

	// Read AGENTS.md
	if agentsFile != "" {
		agents, err := os.ReadFile(agentsFile)
		if err == nil && len(agents) > 0 {
			p.sections = append(p.sections, string(agents))
		}
	}

	// Read STYLE.md
	if styleFile != "" {
		style, err := os.ReadFile(styleFile)
		if err == nil && len(style) > 0 {
			p.sections = append(p.sections, string(style))
		}
	}

	if err := p.write(); err != nil {
		return nil, err
	}
	return p, nil
}

// Path returns the filesystem path to the assembled prompt.
func (p *Prompt) Path() string {
	return p.path
}

// AppendRetryContext adds error output to the prompt for a retry attempt.
func (p *Prompt) AppendRetryContext(verifyErr error, output string) error {
	section := fmt.Sprintf(`---

## Retry Context

The previous attempt failed verification. Here is the error output:

%s

%s

Fix the issues above and re-run verification. Do NOT repeat the same mistakes.`,
		"```",
		output)
	section += "\n```\n"

	// Actually build the section properly
	section = fmt.Sprintf(`---

## Retry Context

The previous attempt failed verification. Here is the error output:

`+"```"+`
%s
`+"```"+`

Error: %v

Fix the issues above and re-run verification. Do NOT repeat the same mistakes.`,
		output, verifyErr)

	p.sections = append(p.sections, section)
	return p.write()
}

// AppendReviewFeedback adds review issues to the prompt for fixing.
func (p *Prompt) AppendReviewFeedback(issues string) error {
	section := fmt.Sprintf(`---

## Review Feedback

The code review found these issues that must be fixed:

%s

Address every issue listed above. After fixing, run verification again.`, issues)

	p.sections = append(p.sections, section)
	return p.write()
}

func (p *Prompt) write() error {
	if err := os.MkdirAll(p.dir, 0o755); err != nil {
		return fmt.Errorf("creating prompt dir: %w", err)
	}
	content := strings.Join(p.sections, "\n\n")
	return os.WriteFile(p.path, []byte(content), 0o644)
}
