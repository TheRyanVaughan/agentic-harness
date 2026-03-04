package templates

import "embed"

//go:embed AGENTS.md STYLE.md spec.template.md justfile
var FS embed.FS
