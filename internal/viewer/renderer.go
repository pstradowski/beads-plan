package viewer

import (
	"fmt"
	"strings"
	"time"
)

// RenderOptions controls the markdown output.
type RenderOptions struct {
	ShowBeadIDs bool
	ShowTiers   bool
}

// DefaultRenderOptions returns sensible defaults.
func DefaultRenderOptions() RenderOptions {
	return RenderOptions{
		ShowBeadIDs: true,
		ShowTiers:   true,
	}
}

// RenderTasksMd generates an OpenSpec-compatible tasks.md from an EpicTree.
func RenderTasksMd(tree *EpicTree, opts RenderOptions) string {
	var b strings.Builder

	// Header comment
	b.WriteString(fmt.Sprintf("<!-- Generated from beads epic %s on %s -->\n",
		tree.Root.ID, time.Now().Format("2006-01-02 15:04")))
	b.WriteString("<!-- This file is auto-generated. Edit beads, not this file. -->\n\n")

	totalTasks := 0
	completedTasks := 0
	sectionNum := 0

	for _, sub := range tree.SubEpics {
		sectionNum++

		if sub.Bead.Type == "epic" && len(sub.Tasks) > 0 {
			// Section header
			title := cleanTitle(sub.Bead.Title)
			b.WriteString(fmt.Sprintf("## %d. %s\n\n", sectionNum, title))

			for _, task := range sub.Tasks {
				totalTasks++
				checkbox := "[ ]"
				if task.Status == "closed" {
					checkbox = "[x]"
					completedTasks++
				}

				line := fmt.Sprintf("- %s %s", checkbox, cleanTitle(task.Title))

				// Append bead ID and tier tag
				var tags []string
				if opts.ShowBeadIDs {
					tags = append(tags, task.ID)
				}
				if opts.ShowTiers {
					if tier, ok := task.Metadata["tier"]; ok && tier != "" {
						tags = append(tags, tier)
					}
				}
				if len(tags) > 0 {
					line += " <!-- " + strings.Join(tags, " | ") + " -->"
				}

				b.WriteString(line + "\n")
			}
			b.WriteString("\n")
		} else {
			// Collapsed section (single task directly under root)
			totalTasks++
			checkbox := "[ ]"
			if sub.Bead.Status == "closed" {
				checkbox = "[x]"
				completedTasks++
			}
			title := cleanTitle(sub.Bead.Title)
			b.WriteString(fmt.Sprintf("## %d. %s\n\n", sectionNum, title))

			line := fmt.Sprintf("- %s %s", checkbox, title)
			var tags []string
			if opts.ShowBeadIDs {
				tags = append(tags, sub.Bead.ID)
			}
			if opts.ShowTiers {
				if tier, ok := sub.Bead.Metadata["tier"]; ok && tier != "" {
					tags = append(tags, tier)
				}
			}
			if len(tags) > 0 {
				line += " <!-- " + strings.Join(tags, " | ") + " -->"
			}
			b.WriteString(line + "\n\n")
		}
	}

	// Progress footer
	pct := 0
	if totalTasks > 0 {
		pct = completedTasks * 100 / totalTasks
	}
	b.WriteString(fmt.Sprintf("<!-- Progress: %d/%d tasks complete (%d%%) -->\n",
		completedTasks, totalTasks, pct))

	return b.String()
}

// cleanTitle removes common prefixes like "N. " or "N.N " from bead titles.
func cleanTitle(title string) string {
	// Strip leading "N. " pattern (section number prefix)
	parts := strings.SplitN(title, ". ", 2)
	if len(parts) == 2 {
		// Check if the first part looks like a number
		isNum := true
		for _, c := range parts[0] {
			if c < '0' || c > '9' {
				isNum = false
				break
			}
		}
		if isNum {
			return parts[1]
		}
	}

	// Strip leading "N.N " pattern (task number prefix)
	if len(title) > 3 {
		spaceIdx := strings.Index(title, " ")
		if spaceIdx > 0 && spaceIdx < 8 {
			prefix := title[:spaceIdx]
			if strings.Contains(prefix, ".") {
				allNumDot := true
				for _, c := range prefix {
					if (c < '0' || c > '9') && c != '.' {
						allNumDot = false
						break
					}
				}
				if allNumDot {
					return title[spaceIdx+1:]
				}
			}
		}
	}

	return title
}
