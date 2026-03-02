package cli

import (
	"fmt"
	"strings"

	"github.com/pstradowski/beads-plan/internal/config"
	"github.com/spf13/cobra"
)

const skillTemplate = `# beads-plan Agent Skill

## Activation
Use this skill when:
- Working on tasks tracked in beads with tier/parallelism metadata
- Dispatching subtasks to appropriate model tiers
- Generating or refreshing tasks.md from beads

## Commands

### beads-plan plan <change-dir>
Create a beads epic from an OpenSpec change directory.
- Parses tasks.md, proposal.md, design.md, specs/
- Creates nested bead hierarchy: epic → sub-epics → tasks
- Assesses complexity and assigns tier (fast/standard/advanced)
- Analyzes parallelism and creates dependency edges
- Enriches tasks with context, acceptance criteria, output schema

Flags:
- ` + "`--dry-run`" + `: Preview planned structure without creating beads
- ` + "`--profile <name>`" + `: Select provider profile for tier→model resolution
- ` + "`--json`" + `: Output structured JSON

### beads-plan view <epic-id>
Generate tasks.md from a beads epic hierarchy.
- Reads epic recursively via bd show
- Renders checkboxes with bead IDs and tier tags
- Includes progress footer

Flags:
- ` + "`--output <file>`" + `: Write to file (default: stdout)

### beads-plan prime
Output this skill definition.

## Metadata Schema

Each task bead includes structured metadata:

| Field | Values | Description |
|-------|--------|-------------|
| tier | fast, standard, advanced | Capability tier for execution |
| complexity | low, medium, high | Assessed task complexity |
| model | (provider-specific) | Concrete model when profile active |
| parallelism | parallel, sequential, mixed | Execution mode for child tasks |
| parallel_groups | [[id,...], ...] | Groups of concurrent children |
| change | (change name) | OpenSpec change provenance |

## Tier Dispatch

When executing tasks, use the tier to select the appropriate agent:
- **fast**: Simple config, boilerplate, scaffolding tasks
- **standard**: Multi-file changes, integration, testing
- **advanced**: Architecture, refactoring, cross-cutting concerns

## Task Output Protocol

After completing a task, record in metadata:
- **files_changed**: List of file paths created or modified
- **decisions**: Architectural or implementation decisions made
- **discoveries**: Unexpected findings or issues encountered

## Parallelism

Check parent bead metadata for parallelism hints:
- **parallel**: All children can run concurrently
- **sequential**: Execute children in order
- **mixed**: Consult parallel_groups for wave scheduling
{{PROFILE_SECTION}}
`

var primeCmd = &cobra.Command{
	Use:   "prime",
	Short: "Emit agent skill definition",
	Long:  "Output a SKILL.md file that teaches a coding agent how to use beads-plan, interpret metadata fields, and dispatch subagents using parallelism hints.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, _ := config.LoadDefault()

		profileSection := ""
		if cfg != nil && len(cfg.Profiles) > 0 {
			var b strings.Builder
			b.WriteString("\n## Provider Profiles\n\nConfigured profiles:\n\n")
			for name, p := range cfg.Profiles {
				b.WriteString(fmt.Sprintf("### %s\n", name))
				b.WriteString(fmt.Sprintf("- fast → %s\n", orDefaultPrime(p.Fast, "(not set)")))
				b.WriteString(fmt.Sprintf("- standard → %s\n", orDefaultPrime(p.Standard, "(not set)")))
				b.WriteString(fmt.Sprintf("- advanced → %s\n\n", orDefaultPrime(p.Advanced, "(not set)")))
			}
			if cfg.DefaultProfile != "" {
				b.WriteString(fmt.Sprintf("Default profile: %s\n", cfg.DefaultProfile))
			}
			profileSection = b.String()
		}

		output := strings.Replace(skillTemplate, "{{PROFILE_SECTION}}", profileSection, 1)
		fmt.Print(output)
		return nil
	},
}

func orDefaultPrime(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
