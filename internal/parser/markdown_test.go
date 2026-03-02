package parser

import (
	"testing"
)

func TestParseStandardTasksMd(t *testing.T) {
	input := `## 1. Database Migrations

- [ ] 1.1 Create migration for doc schema
- [x] 1.2 Grant reader role permissions

## 2. Service Scaffold

- [ ] 2.1 Create project structure
- [ ] 2.2 Add dependencies
- [x] 2.3 Write Dockerfile
`
	tree, err := ParseTasksMarkdown(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tree.Sections) != 2 {
		t.Fatalf("expected 2 sections, got %d", len(tree.Sections))
	}

	s1 := tree.Sections[0]
	if s1.Number != "1" || s1.Title != "Database Migrations" {
		t.Errorf("section 1: got number=%q title=%q", s1.Number, s1.Title)
	}
	if len(s1.Tasks) != 2 {
		t.Fatalf("section 1: expected 2 tasks, got %d", len(s1.Tasks))
	}
	if s1.Tasks[0].IsCompleted {
		t.Error("task 1.1 should not be completed")
	}
	if !s1.Tasks[1].IsCompleted {
		t.Error("task 1.2 should be completed")
	}

	s2 := tree.Sections[1]
	if s2.Number != "2" {
		t.Errorf("section 2: got number=%q", s2.Number)
	}
	if len(s2.Tasks) != 3 {
		t.Fatalf("section 2: expected 3 tasks, got %d", len(s2.Tasks))
	}
	if tree.TotalTasks() != 5 {
		t.Errorf("expected 5 total tasks, got %d", tree.TotalTasks())
	}
	if tree.CompletedTasks() != 2 {
		t.Errorf("expected 2 completed tasks, got %d", tree.CompletedTasks())
	}
}

func TestParseFlatTasksMd(t *testing.T) {
	input := `- [ ] 1.1 Do first thing
- [x] 1.2 Do second thing
- [ ] 1.3 Do third thing
`
	tree, err := ParseTasksMarkdown(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tree.Sections) != 1 {
		t.Fatalf("expected 1 section for flat file, got %d", len(tree.Sections))
	}
	if tree.Sections[0].Title != "Tasks" {
		t.Errorf("expected default title 'Tasks', got %q", tree.Sections[0].Title)
	}
	if len(tree.Sections[0].Tasks) != 3 {
		t.Errorf("expected 3 tasks, got %d", len(tree.Sections[0].Tasks))
	}
}

func TestParseEmptyInput(t *testing.T) {
	_, err := ParseTasksMarkdown("")
	if err == nil {
		t.Error("expected error for empty input")
	}
}

func TestParseNoTasks(t *testing.T) {
	input := `## 1. Empty Section

Some text that isn't a checkbox.

## 2. Also Empty
`
	_, err := ParseTasksMarkdown(input)
	if err == nil {
		t.Error("expected error when no tasks found")
	}
}

func TestParseUppercaseX(t *testing.T) {
	input := `## 1. Test

- [X] 1.1 Completed with uppercase X
- [ ] 1.2 Not completed
`
	tree, err := ParseTasksMarkdown(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !tree.Sections[0].Tasks[0].IsCompleted {
		t.Error("uppercase X should be treated as completed")
	}
}

func TestParseSkipsNonCheckboxLines(t *testing.T) {
	input := `## 1. Mixed Content

Some description text.

- [ ] 1.1 Real task
- This is not a checkbox
- [ ] 1.2 Another real task

More text.
`
	tree, err := ParseTasksMarkdown(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tree.Sections[0].Tasks) != 2 {
		t.Errorf("expected 2 tasks (skipping non-checkbox), got %d", len(tree.Sections[0].Tasks))
	}
}

func TestParsePreservesTaskTitle(t *testing.T) {
	input := `## 1. Test

- [ ] 1.1 Create migration for doc schema with complex SQL and multiple tables
`
	tree, err := ParseTasksMarkdown(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "Create migration for doc schema with complex SQL and multiple tables"
	if tree.Sections[0].Tasks[0].Title != want {
		t.Errorf("got title %q, want %q", tree.Sections[0].Tasks[0].Title, want)
	}
}
