package planner

import (
	"strings"
)

// Complexity represents the assessed complexity of a task.
type Complexity string

const (
	ComplexityLow    Complexity = "low"
	ComplexityMedium Complexity = "medium"
	ComplexityHigh   Complexity = "high"
)

// Keywords that suggest different complexity levels.
var (
	highKeywords = []string{
		"architect", "refactor", "redesign", "migration", "cross-cutting",
		"breaking change", "security", "authentication", "authorization",
		"distributed", "consensus", "orchestrat",
	}
	lowKeywords = []string{
		"config", "configuration", "rename", "typo", "comment",
		"boilerplate", "scaffold", "stub", "placeholder", "gitignore",
		"readme", "license", "version bump", "update dependency",
	}
)

// AssessComplexity evaluates a task's complexity based on heuristics applied
// to the task title, and optionally spec/design context.
func AssessComplexity(title string, specContext string, designContext string) Complexity {
	combined := strings.ToLower(title + " " + specContext + " " + designContext)

	// Check for high-complexity indicators first
	highScore := 0
	for _, kw := range highKeywords {
		if strings.Contains(combined, kw) {
			highScore++
		}
	}
	if highScore >= 2 {
		return ComplexityHigh
	}

	// Check for low-complexity indicators
	lowScore := 0
	for _, kw := range lowKeywords {
		if strings.Contains(combined, kw) {
			lowScore++
		}
	}

	// Count integration signals in the full text
	integrationTerms := []string{
		"integrate", "integration", "api", "endpoint", "database",
		"service", "client", "server", "handler", "middleware",
		"test", "tests", "testing",
	}
	integrationScore := 0
	for _, term := range integrationTerms {
		if strings.Contains(combined, term) {
			integrationScore++
		}
	}

	// If strong low signals and no integration signals, it's low
	if lowScore >= 1 && integrationScore == 0 && highScore == 0 {
		return ComplexityLow
	}

	// Single high keyword or multiple integration terms → high
	if highScore >= 1 && integrationScore >= 2 {
		return ComplexityHigh
	}

	// Default: medium covers most real tasks
	return ComplexityMedium
}
