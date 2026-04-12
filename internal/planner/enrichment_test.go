package planner

import (
	"strings"
	"testing"

	"github.com/pstradowski/beads-plan/internal/parser"
)

func TestDetectTestExplicitMarker(t *testing.T) {
	section := parser.Section{Number: "2", Title: "Implementation"}
	task := parser.Task{Number: "2.3", Title: "Implement parser <!-- test -->"}

	if DetectTest(task, section) == "" {
		t.Error("explicit <!-- test --> marker should classify as test even in a non-test section")
	}
}

func TestDetectTestExplicitMarkerWhitespaceTolerant(t *testing.T) {
	section := parser.Section{Number: "2", Title: "Implementation"}
	task := parser.Task{Number: "2.3", Title: "Task title <!--   test  -->"}

	if DetectTest(task, section) == "" {
		t.Error("explicit marker detection should be whitespace-tolerant inside the comment")
	}
}

func TestDetectTestSectionTitle(t *testing.T) {
	cases := []struct {
		name    string
		section string
	}{
		{"plural", "Tests"},
		{"singular", "Test"},
		{"gerund", "Testing"},
		{"compound", "5. Unit tests"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			section := parser.Section{Number: "5", Title: tc.section}
			task := parser.Task{Number: "5.1", Title: "Add coverage for parser"}
			if DetectTest(task, section) == "" {
				t.Errorf("section title %q should classify its tasks as tests", tc.section)
			}
		})
	}
}

func TestDetectTestKeywordFallback(t *testing.T) {
	section := parser.Section{Number: "3", Title: "Implementation"}
	task := parser.Task{Number: "3.4", Title: "Add unit tests for the parser"}

	if DetectTest(task, section) == "" {
		t.Error("keyword fallback should classify obvious test tasks inside non-test sections")
	}
}

func TestDetectTestKeywordSuppressedInNonTestSection(t *testing.T) {
	cases := []struct {
		name    string
		section string
		title   string
	}{
		{"refactor", "Refactor", "Test that the new interface compiles"},
		{"document", "Document", "Test that docs render"},
		{"docs", "Docs", "Test the doc build"},
		{"rename", "Rename", "Test no callers break"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			section := parser.Section{Number: "4", Title: tc.section}
			task := parser.Task{Number: "4.1", Title: tc.title}
			if DetectTest(task, section) != "" {
				t.Errorf("section %q with task %q: keyword fallback must NOT fire in clearly non-test sections", tc.section, tc.title)
			}
		})
	}
}

func TestDetectTestExplicitMarkerMustBeTrailing(t *testing.T) {
	// An inline reference to the marker syntax inside the task
	// title (e.g. inside a code span) should NOT be interpreted as
	// the explicit marker. Only a trailing annotation counts.
	section := parser.Section{Title: "Implementation"}
	task := parser.Task{
		Title: "Implement the explicit-marker rule: match `<!-- test -->` anywhere",
	}
	if got := DetectTest(task, section); got == "explicit" {
		t.Errorf("title with mid-string marker reference should not trigger the explicit rule, got %q", got)
	}
}

func TestDetectTestIgnoresTestInsideHTMLComment(t *testing.T) {
	// A task that documents the explicit-marker syntax contains the
	// substring <!-- test --> inside a code span. The keyword rule
	// must not read that "test" as a standalone occurrence.
	section := parser.Section{Title: "Implementation"}
	task := parser.Task{
		Title: "Implement the explicit-marker rule: match `<!-- test -->` anywhere",
	}
	if got := DetectTest(task, section); got != "" {
		t.Errorf("test inside an HTML comment should be stripped before keyword scan, got %q", got)
	}
}

func TestDetectTestRejectsColonCompounds(t *testing.T) {
	// Compound identifiers like meow:test, config:test, dev:test
	// should not match the keyword rule.
	section := parser.Section{Title: "Formula authoring"}
	task := parser.Task{
		Title: "Apply labels `meow:test`, `meow:test-run`, `meow:test-correct` to beads",
	}
	if got := DetectTest(task, section); got != "" {
		t.Errorf("title containing only colon-prefixed test compounds should not be detected, got %q", got)
	}
}

func TestDetectTestRejectsHyphenatedCompounds(t *testing.T) {
	// These section/task titles mention test-X or X-test compounds
	// but are not themselves about writing tests. The detector must
	// reject them so that sections about the test system (compile
	// branch, detection rules, retry pattern) don't get mistakenly
	// wrapped in test-task sub-epics.
	cases := []struct {
		name    string
		section string
		title   string
	}{
		{"test-task in section", "Test-task detection", "Implement the branch"},
		{"test-task in title", "Implementation", "Emit a four-bead test-task sub-epic"},
		{"smoke-test compound", "Scaffolding", "Create a smoke-test fixture directory"},
		{"run-tests-N compound", "Implementation", "Collect leaf IDs excluding run-tests-N and correct-N beads"},
		{"stuck-test compound", "Formula authoring", "Implement the meow:stuck-test escalation path"},
		{"tested substring", "Setup", "Everything is tested manually"},
		{"testimony substring", "Setup", "Gather testimony from the oracle"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			section := parser.Section{Title: tc.section}
			task := parser.Task{Title: tc.title}
			if got := DetectTest(task, section); got != "" {
				t.Errorf("section=%q title=%q: expected no detection, got %q", tc.section, tc.title, got)
			}
		})
	}
}

func TestDetectTestRegularTaskIsNotTest(t *testing.T) {
	section := parser.Section{Number: "1", Title: "Setup"}
	task := parser.Task{Number: "1.1", Title: "Create directory structure"}

	if DetectTest(task, section) != "" {
		t.Error("a task with no test indicators should not be classified as a test task")
	}
}

func TestDetectTestReturnsRuleName(t *testing.T) {
	cases := []struct {
		name    string
		section parser.Section
		task    parser.Task
		want    string
	}{
		{"explicit wins over section", parser.Section{Title: "Tests"}, parser.Task{Title: "foo <!-- test -->"}, "explicit"},
		{"section rule", parser.Section{Title: "Tests"}, parser.Task{Title: "plain task"}, "section"},
		{"keyword fallback", parser.Section{Title: "Implementation"}, parser.Task{Title: "add unit tests"}, "keyword"},
		{"none", parser.Section{Title: "Setup"}, parser.Task{Title: "create dir"}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := DetectTest(tc.task, tc.section)
			if got != tc.want {
				t.Errorf("DetectTest(%+v, %+v) = %q, want %q", tc.task, tc.section, got, tc.want)
			}
		})
	}
}

func TestStripTestMarker(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"Add unit tests <!-- test -->", "Add unit tests"},
		{"Normal task", "Normal task"},
		{"Trailing marker <!--  test  -->", "Trailing marker"},
	}
	for _, tc := range cases {
		got := StripTestMarker(tc.in)
		if got != tc.want {
			t.Errorf("StripTestMarker(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

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
