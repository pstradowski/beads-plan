package planner

import (
	"fmt"
	"strings"

	"github.com/pstradowski/beads-plan/internal/parser"
)

const defaultMaxWords = 500

const taskOutputSchema = `## Expected Task Output

When completing this task, capture the following in task metadata:
- files_changed: list of file paths created or modified
- decisions: architectural or implementation decisions made
- discoveries: unexpected findings or issues encountered`

// EnrichTasks takes parsed tasks and artifacts, returning enriched tasks
// with expanded descriptions, design context, spec references, acceptance
// criteria, output schema, and tier assessment.
func EnrichTasks(sections []parser.Section, artifacts *parser.Artifacts, profile interface{ ResolveModel(string) string }) map[string]EnrichedTask {
	enriched := make(map[string]EnrichedTask)

	for _, s := range sections {
		for _, task := range s.Tasks {
			e := EnrichedTask{Task: task}

			// Context distribution: extract relevant content
			e.Description = buildDescription(task, s, artifacts)
			e.Design = extractDesignContext(task, artifacts)
			e.SpecID = matchSpec(task, s, artifacts)
			e.Acceptance = extractAcceptance(task, artifacts)

			// Output schema injection
			e.Notes = taskOutputSchema

			// Description size guard
			guardDescriptionSize(&e, defaultMaxWords)

			// Tier assessment
			assignment := AssignTier(task.Title, e.SpecID, e.Design, nil)
			e.Tier = string(assignment.Tier)

			enriched[task.Number] = e
		}
	}

	return enriched
}

// buildDescription creates an expanded description from the task title
// and relevant proposal/spec context.
func buildDescription(task parser.Task, section parser.Section, artifacts *parser.Artifacts) string {
	var parts []string

	parts = append(parts, fmt.Sprintf("Task %s: %s", task.Number, task.Title))
	parts = append(parts, fmt.Sprintf("Section: %s. %s", section.Number, section.Title))

	// Extract relevant proposal paragraphs
	if artifacts != nil && artifacts.Proposal != "" {
		keywords := extractKeywords(task.Title)
		relevant := extractRelevantParagraphs(artifacts.Proposal, keywords)
		if relevant != "" {
			parts = append(parts, "From proposal:\n"+relevant)
		}
	}

	return strings.Join(parts, "\n\n")
}

// extractDesignContext finds design decisions relevant to the task.
func extractDesignContext(task parser.Task, artifacts *parser.Artifacts) string {
	if artifacts == nil || artifacts.Design == "" {
		return ""
	}

	keywords := extractKeywords(task.Title)
	return extractRelevantParagraphs(artifacts.Design, keywords)
}

// matchSpec finds the most relevant spec file for a task.
func matchSpec(task parser.Task, section parser.Section, artifacts *parser.Artifacts) string {
	if artifacts == nil || len(artifacts.Specs) == 0 {
		return ""
	}

	titleLower := strings.ToLower(task.Title + " " + section.Title)
	bestMatch := ""
	bestScore := 0

	for _, spec := range artifacts.Specs {
		capWords := strings.Split(spec.Capability, "-")
		score := 0
		for _, w := range capWords {
			if len(w) >= 3 && strings.Contains(titleLower, w) {
				score++
			}
		}
		if score > bestScore {
			bestScore = score
			bestMatch = spec.Path
		}
	}

	return bestMatch
}

// extractAcceptance finds WHEN/THEN patterns in specs that relate to the task.
func extractAcceptance(task parser.Task, artifacts *parser.Artifacts) string {
	if artifacts == nil || len(artifacts.Specs) == 0 {
		return ""
	}

	keywords := extractKeywords(task.Title)
	var criteria []string

	for _, spec := range artifacts.Specs {
		lines := strings.Split(spec.Content, "\n")
		for i, line := range lines {
			lower := strings.ToLower(line)
			isScenario := strings.Contains(lower, "when") || strings.Contains(lower, "then") ||
				strings.Contains(lower, "given") || strings.HasPrefix(strings.TrimSpace(lower), "- ")

			if !isScenario {
				continue
			}

			// Check if this line relates to the task
			for _, kw := range keywords {
				if len(kw) >= 4 && strings.Contains(lower, kw) {
					criteria = append(criteria, strings.TrimSpace(lines[i]))
					break
				}
			}
		}
	}

	if len(criteria) == 0 {
		return ""
	}
	return strings.Join(criteria, "\n")
}

// extractRelevantParagraphs finds paragraphs containing any of the keywords.
func extractRelevantParagraphs(text string, keywords []string) string {
	paragraphs := strings.Split(text, "\n\n")
	var relevant []string

	for _, p := range paragraphs {
		pLower := strings.ToLower(p)
		for _, kw := range keywords {
			if len(kw) >= 4 && strings.Contains(pLower, kw) {
				relevant = append(relevant, strings.TrimSpace(p))
				break
			}
		}
	}

	return strings.Join(relevant, "\n\n")
}

// guardDescriptionSize truncates descriptions exceeding maxWords and appends
// a spec reference for full details.
func guardDescriptionSize(task *EnrichedTask, maxWords int) {
	words := strings.Fields(task.Description)
	if len(words) <= maxWords {
		return
	}

	truncated := strings.Join(words[:maxWords], " ")
	ref := ""
	if task.SpecID != "" {
		ref = fmt.Sprintf("\n\n[Truncated — see %s for full details]", task.SpecID)
	} else {
		ref = "\n\n[Truncated — see spec files for full details]"
	}
	task.Description = truncated + ref
}
