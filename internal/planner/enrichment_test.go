package planner

import (
	"strings"
	"testing"

	"github.com/pstradowski/beads-plan/internal/parser"
)

func TestBuildDescription(t *testing.T) {
	task := parser.Task{Number: "1.1", Title: "Create database migration"}
	section := parser.Section{Number: "1", Title: "Database Setup"}
	artifacts := &parser.Artifacts{
		Proposal: "## Why\n\nWe need a database migration to add the users table.\n\n## Impact\n\nNo impact on existing services.",
	}

	desc := buildDescription(task, section, artifacts)
	if !strings.Contains(desc, "Task 1.1") {
		t.Error("description should contain task number")
	}
	if !strings.Contains(desc, "database migration") {
		t.Error("description should contain relevant proposal content")
	}
}

func TestBuildDescriptionNoArtifacts(t *testing.T) {
	task := parser.Task{Number: "1.1", Title: "Do something"}
	section := parser.Section{Number: "1", Title: "Section"}

	desc := buildDescription(task, section, nil)
	if !strings.Contains(desc, "Task 1.1") {
		t.Error("description should still contain task number without artifacts")
	}
}

func TestExtractDesignContext(t *testing.T) {
	task := parser.Task{Number: "1.1", Title: "Implement authentication handler"}
	artifacts := &parser.Artifacts{
		Design: "## Decision 1: Database\n\nUse PostgreSQL for storage.\n\n## Decision 2: Authentication\n\nUse JWT tokens for authentication with refresh tokens.",
	}

	design := extractDesignContext(task, artifacts)
	if !strings.Contains(design, "Authentication") {
		t.Error("should extract authentication-related design context")
	}
}

func TestExtractDesignContextEmpty(t *testing.T) {
	task := parser.Task{Number: "1.1", Title: "Do thing"}
	design := extractDesignContext(task, nil)
	if design != "" {
		t.Error("expected empty design for nil artifacts")
	}
}

func TestMatchSpec(t *testing.T) {
	task := parser.Task{Number: "1.1", Title: "Implement plan command"}
	section := parser.Section{Number: "1", Title: "CLI"}
	artifacts := &parser.Artifacts{
		Specs: []parser.SpecFile{
			{Capability: "beads-plan-cli", Path: "specs/beads-plan-cli/spec.md", Content: "CLI spec"},
			{Capability: "beads-tasks-view", Path: "specs/beads-tasks-view/spec.md", Content: "View spec"},
		},
	}

	specID := matchSpec(task, section, artifacts)
	if specID != "specs/beads-plan-cli/spec.md" {
		t.Errorf("expected CLI spec match, got %s", specID)
	}
}

func TestMatchSpecNoMatch(t *testing.T) {
	task := parser.Task{Number: "1.1", Title: "Unrelated thing"}
	section := parser.Section{Number: "1", Title: "Other"}
	artifacts := &parser.Artifacts{
		Specs: []parser.SpecFile{
			{Capability: "specific-feature", Path: "specs/specific-feature/spec.md"},
		},
	}

	specID := matchSpec(task, section, artifacts)
	// May return empty or best-effort match
	_ = specID // no assertion — just shouldn't panic
}

func TestExtractAcceptance(t *testing.T) {
	task := parser.Task{Number: "1.1", Title: "Implement parser for tasks"}
	artifacts := &parser.Artifacts{
		Specs: []parser.SpecFile{
			{
				Capability: "parser",
				Content:    "## Requirements\n\n- WHEN a valid tasks.md is provided THEN parser returns TaskTree\n- WHEN input is empty THEN parser returns error\n- Unrelated requirement about logging",
			},
		},
	}

	acceptance := extractAcceptance(task, artifacts)
	if !strings.Contains(acceptance, "tasks") {
		t.Error("should extract task-related acceptance criteria")
	}
}

func TestExtractAcceptanceNoSpecs(t *testing.T) {
	task := parser.Task{Number: "1.1", Title: "Something"}
	acceptance := extractAcceptance(task, nil)
	if acceptance != "" {
		t.Error("expected empty acceptance for nil artifacts")
	}
}

func TestGuardDescriptionSizeUnderLimit(t *testing.T) {
	task := &EnrichedTask{
		Description: "Short description here",
	}
	guardDescriptionSize(task, 500)
	if strings.Contains(task.Description, "Truncated") {
		t.Error("should not truncate short descriptions")
	}
}

func TestGuardDescriptionSizeOverLimit(t *testing.T) {
	// Build a 600-word description
	words := make([]string, 600)
	for i := range words {
		words[i] = "word"
	}
	task := &EnrichedTask{
		Description: strings.Join(words, " "),
		SpecID:      "specs/test/spec.md",
	}
	guardDescriptionSize(task, 500)
	if !strings.Contains(task.Description, "Truncated") {
		t.Error("should truncate over-limit descriptions")
	}
	if !strings.Contains(task.Description, "specs/test/spec.md") {
		t.Error("should include spec reference in truncation notice")
	}
	// Count words before the truncation notice
	truncParts := strings.SplitN(task.Description, "\n\n[Truncated", 2)
	wordCount := len(strings.Fields(truncParts[0]))
	if wordCount > 500 {
		t.Errorf("truncated description should have ≤500 words, got %d", wordCount)
	}
}

func TestEnrichTasks(t *testing.T) {
	sections := []parser.Section{
		{Number: "1", Title: "Database", Tasks: []parser.Task{
			{Number: "1.1", Title: "Create migration"},
			{Number: "1.2", Title: "Write handler"},
		}},
	}
	artifacts := &parser.Artifacts{
		Proposal: "## Why\n\nNeed database migration for users.",
		Design:   "## Decision\n\nUse PostgreSQL.",
		Specs: []parser.SpecFile{
			{Capability: "database", Path: "specs/database/spec.md", Content: "DB spec"},
		},
	}

	enriched := EnrichTasks(sections, artifacts, nil)
	if len(enriched) != 2 {
		t.Fatalf("expected 2 enriched tasks, got %d", len(enriched))
	}
	e := enriched["1.1"]
	if e.Description == "" {
		t.Error("expected non-empty description")
	}
	if e.Notes == "" {
		t.Error("expected task output schema in notes")
	}
	if !strings.Contains(e.Notes, "files_changed") {
		t.Error("notes should contain task output schema")
	}
	if e.Tier == "" {
		t.Error("expected tier assignment")
	}
}
