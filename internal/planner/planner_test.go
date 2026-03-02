package planner

import (
	"fmt"
	"testing"

	"github.com/pstradowski/beads-plan/internal/parser"
)

// mockClient records calls and returns sequential IDs.
type mockClient struct {
	creates []CreateOpts
	deps    [][2]string // [from, to] pairs
	closes  [][2]string // [id, msg] pairs
	nextID  int
}

func (m *mockClient) Create(opts CreateOpts) (string, error) {
	m.creates = append(m.creates, opts)
	m.nextID++
	return fmt.Sprintf("BEAD-%d", m.nextID), nil
}

func (m *mockClient) AddDep(issueID, dependsOnID string) error {
	m.deps = append(m.deps, [2]string{issueID, dependsOnID})
	return nil
}

func (m *mockClient) Close(issueID, message string) error {
	m.closes = append(m.closes, [2]string{issueID, message})
	return nil
}

func TestCreateRootEpic(t *testing.T) {
	mc := &mockClient{}
	p := &Planner{Client: mc, ChangeName: "my-change"}

	id, err := p.CreateRootEpic()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id == "" {
		t.Error("expected non-empty root ID")
	}
	if len(mc.creates) != 1 {
		t.Fatalf("expected 1 create call, got %d", len(mc.creates))
	}
	c := mc.creates[0]
	if c.Type != "epic" {
		t.Errorf("expected type=epic, got %s", c.Type)
	}
	if c.Metadata["change"] != "my-change" {
		t.Errorf("expected metadata change=my-change, got %s", c.Metadata["change"])
	}
}

func TestCreateSubEpicsMultiTask(t *testing.T) {
	mc := &mockClient{}
	p := &Planner{Client: mc, ChangeName: "test"}

	sections := []parser.Section{
		{Number: "1", Title: "Setup", Tasks: []parser.Task{
			{Number: "1.1", Title: "First"},
			{Number: "1.2", Title: "Second"},
		}},
	}

	ids, err := p.CreateSubEpics("ROOT", sections)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 1 {
		t.Fatalf("expected 1 sub-epic, got %d", len(ids))
	}
	// Multi-task section should create an epic
	if mc.creates[0].Type != "epic" {
		t.Errorf("expected type=epic for multi-task section, got %s", mc.creates[0].Type)
	}
	if mc.creates[0].Parent != "ROOT" {
		t.Errorf("expected parent=ROOT, got %s", mc.creates[0].Parent)
	}
}

func TestCreateSubEpicsCollapse(t *testing.T) {
	mc := &mockClient{}
	p := &Planner{Client: mc, ChangeName: "test"}

	sections := []parser.Section{
		{Number: "1", Title: "Solo", Tasks: []parser.Task{
			{Number: "1.1", Title: "Only task"},
		}},
	}

	ids, err := p.CreateSubEpics("ROOT", sections)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 1 {
		t.Fatalf("expected 1 ID, got %d", len(ids))
	}
	// Single-task section should collapse to a task, not epic
	if mc.creates[0].Type != "task" {
		t.Errorf("expected type=task for collapsed section, got %s", mc.creates[0].Type)
	}
}

func TestCreateLeafTasks(t *testing.T) {
	mc := &mockClient{}
	p := &Planner{Client: mc, ChangeName: "test"}

	sections := []parser.Section{
		{Number: "1", Title: "Setup", Tasks: []parser.Task{
			{Number: "1.1", Title: "First"},
			{Number: "1.2", Title: "Second"},
		}},
	}
	subEpicIDs := map[int]string{0: "EPIC-1"}
	enriched := map[string]EnrichedTask{
		"1.1": {
			Task:        parser.Task{Number: "1.1", Title: "First"},
			Description: "Do the first thing",
			Tier:        "fast",
		},
	}

	ids, err := p.CreateLeafTasks(subEpicIDs, sections, enriched)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 task IDs, got %d", len(ids))
	}

	// Check enrichment was applied to first task
	c1 := mc.creates[0]
	if c1.Description != "Do the first thing" {
		t.Errorf("expected description from enrichment, got %q", c1.Description)
	}
	if c1.Metadata["tier"] != "fast" {
		t.Errorf("expected tier=fast, got %s", c1.Metadata["tier"])
	}
}

func TestCreateLeafTasksSkipsCollapsed(t *testing.T) {
	mc := &mockClient{}
	p := &Planner{Client: mc, ChangeName: "test"}

	sections := []parser.Section{
		{Number: "1", Title: "Solo", Tasks: []parser.Task{
			{Number: "1.1", Title: "Only"},
		}},
	}
	subEpicIDs := map[int]string{0: "COLLAPSED-1"}

	ids, err := p.CreateLeafTasks(subEpicIDs, sections, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Collapsed section: no new creates, ID reused from sub-epic step
	if len(mc.creates) != 0 {
		t.Errorf("expected 0 creates for collapsed section, got %d", len(mc.creates))
	}
	if ids["1.1"] != "COLLAPSED-1" {
		t.Errorf("expected collapsed ID, got %s", ids["1.1"])
	}
}

func TestCreateDependencies(t *testing.T) {
	mc := &mockClient{}
	p := &Planner{Client: mc, ChangeName: "test"}

	taskIDs := map[string]string{
		"1.1": "BEAD-A",
		"1.2": "BEAD-B",
		"2.1": "BEAD-C",
	}
	edges := []DepEdge{
		{From: "1.2", To: "1.1"},
		{From: "2.1", To: "1.2"},
		{From: "9.9", To: "1.1"}, // unknown source, should be skipped
	}

	created, err := p.CreateDependencies(taskIDs, edges)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created != 2 {
		t.Errorf("expected 2 deps created, got %d", created)
	}
	if len(mc.deps) != 2 {
		t.Fatalf("expected 2 dep calls, got %d", len(mc.deps))
	}
	if mc.deps[0] != [2]string{"BEAD-B", "BEAD-A"} {
		t.Errorf("dep 0: got %v", mc.deps[0])
	}
	if mc.deps[1] != [2]string{"BEAD-C", "BEAD-B"} {
		t.Errorf("dep 1: got %v", mc.deps[1])
	}
}

func TestCreateRootEpicDefaultPriority(t *testing.T) {
	mc := &mockClient{}
	p := &Planner{Client: mc, ChangeName: "test"}

	_, err := p.CreateRootEpic()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mc.creates[0].Priority != "P1" {
		t.Errorf("expected default priority P1, got %s", mc.creates[0].Priority)
	}
}

func TestCreateRootEpicCustomPriority(t *testing.T) {
	mc := &mockClient{}
	p := &Planner{Client: mc, ChangeName: "test", Priority: "P0"}

	_, err := p.CreateRootEpic()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mc.creates[0].Priority != "P0" {
		t.Errorf("expected priority P0, got %s", mc.creates[0].Priority)
	}
}
