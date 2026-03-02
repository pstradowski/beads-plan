package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SpecFile holds the path and content of a single spec file.
type SpecFile struct {
	Capability string // e.g., "beads-planning-agent"
	Path       string // relative path, e.g., "specs/beads-planning-agent/spec.md"
	Content    string
}

// Artifacts holds all OpenSpec artifacts from a change directory.
type Artifacts struct {
	ChangeName string
	ChangeDir  string
	Proposal   string     // proposal.md content (may be empty)
	Design     string     // design.md content (may be empty)
	Tasks      string     // tasks.md content
	Specs      []SpecFile // spec files
}

// LoadArtifacts reads all OpenSpec artifacts from a change directory.
func LoadArtifacts(changeDir string) (*Artifacts, error) {
	a := &Artifacts{
		ChangeName: filepath.Base(changeDir),
		ChangeDir:  changeDir,
	}

	// Tasks.md is required
	tasksPath := filepath.Join(changeDir, "tasks.md")
	tasksContent, err := os.ReadFile(tasksPath)
	if err != nil {
		return nil, fmt.Errorf("tasks.md not found in change %s. Run openspec to create it first", a.ChangeName)
	}
	a.Tasks = string(tasksContent)

	// Proposal is optional
	if content, err := os.ReadFile(filepath.Join(changeDir, "proposal.md")); err == nil {
		a.Proposal = string(content)
	}

	// Design is optional
	if content, err := os.ReadFile(filepath.Join(changeDir, "design.md")); err == nil {
		a.Design = string(content)
	}

	// Specs: glob for specs/*/spec.md
	specPattern := filepath.Join(changeDir, "specs", "*", "spec.md")
	matches, _ := filepath.Glob(specPattern)
	for _, m := range matches {
		content, err := os.ReadFile(m)
		if err != nil {
			continue
		}
		// Extract capability name from path: specs/<capability>/spec.md
		rel, _ := filepath.Rel(changeDir, m)
		parts := strings.Split(rel, string(filepath.Separator))
		capability := ""
		if len(parts) >= 2 {
			capability = parts[1] // specs/<capability>/spec.md
		}
		a.Specs = append(a.Specs, SpecFile{
			Capability: capability,
			Path:       rel,
			Content:    string(content),
		})
	}

	return a, nil
}
