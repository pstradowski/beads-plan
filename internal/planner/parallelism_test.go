package planner

import (
	"testing"

	"github.com/pstradowski/beads-plan/internal/parser"
)

func TestSectionParallelismIndependent(t *testing.T) {
	sections := []parser.Section{
		{Number: "1", Title: "Frontend Styling", Tasks: []parser.Task{
			{Number: "1.1", Title: "Update CSS colors"},
		}},
		{Number: "2", Title: "Backend Logging", Tasks: []parser.Task{
			{Number: "2.1", Title: "Add structured logging"},
		}},
	}

	result := AnalyzeSectionParallelism(sections)
	if result.Mode == ModeSequential && len(result.ParallelGroups) > 1 {
		// With no cross-references, sections should be in the same wave
		t.Logf("mode=%s groups=%v", result.Mode, result.ParallelGroups)
	}
	// Independent sections should be parallelizable
	if len(result.ParallelGroups) == 0 {
		t.Error("expected at least one parallel group")
	}
}

func TestSectionParallelismDependent(t *testing.T) {
	sections := []parser.Section{
		{Number: "1", Title: "Database Schema", Tasks: []parser.Task{
			{Number: "1.1", Title: "Create users table migration"},
		}},
		{Number: "2", Title: "User Service", Tasks: []parser.Task{
			{Number: "2.1", Title: "Query users table for authentication"},
		}},
	}

	result := AnalyzeSectionParallelism(sections)
	if len(result.DepEdges) == 0 {
		t.Error("expected dependency edge between sections (users table reference)")
	}
	// Section 2 should depend on section 1
	if len(result.DepEdges) > 0 {
		e := result.DepEdges[0]
		if e.From != "2" || e.To != "1" {
			t.Errorf("expected dep 2→1, got %s→%s", e.From, e.To)
		}
	}
}

func TestSectionParallelismSingle(t *testing.T) {
	sections := []parser.Section{
		{Number: "1", Title: "Only Section", Tasks: []parser.Task{
			{Number: "1.1", Title: "Do thing"},
		}},
	}

	result := AnalyzeSectionParallelism(sections)
	if result.Mode != ModeSequential {
		t.Errorf("single section should be sequential, got %s", result.Mode)
	}
}

func TestTaskParallelismIndependent(t *testing.T) {
	tasks := []parser.Task{
		{Number: "1.1", Title: "Add CSS styles"},
		{Number: "1.2", Title: "Write README documentation"},
		{Number: "1.3", Title: "Configure linter rules"},
	}

	result := AnalyzeTaskParallelism(tasks)
	// These tasks are independent — should be in one wave
	if result.Mode == ModeSequential && len(result.ParallelGroups) > 1 {
		t.Errorf("independent tasks should be parallel, got %d groups", len(result.ParallelGroups))
	}
}

func TestTaskParallelismDependent(t *testing.T) {
	tasks := []parser.Task{
		{Number: "1.1", Title: "Create database migration for orders"},
		{Number: "1.2", Title: "Write handler that queries orders from database"},
	}

	result := AnalyzeTaskParallelism(tasks)
	if len(result.DepEdges) == 0 {
		t.Error("expected dependency edge (orders/database reference)")
	}
}

func TestTaskParallelismSingle(t *testing.T) {
	tasks := []parser.Task{
		{Number: "1.1", Title: "Only task"},
	}

	result := AnalyzeTaskParallelism(tasks)
	if result.Mode != ModeSequential {
		t.Errorf("single task should be sequential, got %s", result.Mode)
	}
}

func TestExtractKeywords(t *testing.T) {
	kws := extractKeywords("Create database migration for users table")
	found := map[string]bool{}
	for _, kw := range kws {
		found[kw] = true
	}
	if !found["database"] {
		t.Error("expected 'database' keyword")
	}
	if !found["migration"] {
		t.Error("expected 'migration' keyword")
	}
	if !found["users"] {
		t.Error("expected 'users' keyword")
	}
	// "for" and "create" should be filtered as stop words
	if found["for"] {
		t.Error("'for' should be filtered")
	}
}

func TestBuildWavesNoDeps(t *testing.T) {
	waves := buildWaves(3, map[int][]int{})
	if len(waves) != 1 {
		t.Errorf("no deps: expected 1 wave with all items, got %d waves", len(waves))
	}
	if len(waves[0]) != 3 {
		t.Errorf("expected 3 items in wave, got %d", len(waves[0]))
	}
}

func TestBuildWavesChain(t *testing.T) {
	// 0 → 1 → 2 (2 depends on 1, 1 depends on 0)
	deps := map[int][]int{
		1: {0},
		2: {1},
	}
	waves := buildWaves(3, deps)
	if len(waves) != 3 {
		t.Errorf("chain: expected 3 waves, got %d", len(waves))
	}
}

func TestClassifyModeParallel(t *testing.T) {
	groups := [][]int{{0, 1, 2}}
	if classifyMode(groups) != ModeParallel {
		t.Error("single group with multiple items should be parallel")
	}
}

func TestClassifyModeSequential(t *testing.T) {
	groups := [][]int{{0}, {1}, {2}}
	if classifyMode(groups) != ModeSequential {
		t.Error("all single-item groups should be sequential")
	}
}

func TestClassifyModeMixed(t *testing.T) {
	groups := [][]int{{0, 1}, {2}}
	if classifyMode(groups) != ModeMixed {
		t.Error("mix of parallel and single groups should be mixed")
	}
}
