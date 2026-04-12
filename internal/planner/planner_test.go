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
	if c.Parent != "" {
		t.Errorf("expected no parent when ParentID is unset, got %q", c.Parent)
	}
}

func TestCreateRootEpicWithParentID(t *testing.T) {
	mc := &mockClient{}
	p := &Planner{Client: mc, ChangeName: "my-change", ParentID: "bd-grandparent"}

	if _, err := p.CreateRootEpic(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mc.creates) != 1 {
		t.Fatalf("expected 1 create call, got %d", len(mc.creates))
	}
	if got := mc.creates[0].Parent; got != "bd-grandparent" {
		t.Errorf("expected parent=bd-grandparent, got %q", got)
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

	ids, err := p.CreateSubEpics("ROOT", sections, nil)
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

	ids, err := p.CreateSubEpics("ROOT", sections, nil)
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

func TestCreateTestTaskSubEpicShape(t *testing.T) {
	mc := &mockClient{}
	p := &Planner{Client: mc, ChangeName: "test"}

	sections := []parser.Section{
		{Number: "8", Title: "Formula smoke tests", Tasks: []parser.Task{
			{Number: "8.1", Title: "Verify formula seed"},
			{Number: "8.2", Title: "Check mandatory gate rejection"},
		}},
	}
	enriched := map[string]EnrichedTask{
		"8.1": {Task: sections[0].Tasks[0], IsTest: true, Tier: "advanced", Description: "do smoke test"},
		"8.2": {Task: sections[0].Tasks[1], IsTest: true, Tier: "standard"},
	}

	subEpicIDs, err := p.CreateSubEpics("ROOT", sections, enriched)
	if err != nil {
		t.Fatalf("sub-epics: %v", err)
	}
	taskIDs, err := p.CreateLeafTasks(subEpicIDs, sections, enriched)
	if err != nil {
		t.Fatalf("leaf tasks: %v", err)
	}

	// One section sub-epic + (per task: 1 test-task epic + 3 children) × 2 = 1 + 8 = 9 creates
	if len(mc.creates) != 9 {
		t.Fatalf("expected 9 create calls (1 sub-epic + 2×4 test beads), got %d", len(mc.creates))
	}

	// The first create is the section sub-epic
	if mc.creates[0].Type != "epic" || mc.creates[0].Parent != "ROOT" {
		t.Errorf("first create should be the section sub-epic, got %+v", mc.creates[0])
	}

	// creates[1..4] = task 8.1 sub-epic (epic + execute + run-tests-1 + correct-1)
	task1Epic := mc.creates[1]
	if task1Epic.Type != "epic" {
		t.Errorf("test-task[8.1] should be type=epic, got %s", task1Epic.Type)
	}
	if !containsLabel(task1Epic.Labels, "meow:test") {
		t.Errorf("test-task[8.1] should carry label meow:test, got %v", task1Epic.Labels)
	}

	execute := mc.creates[2]
	if execute.Description != "do smoke test" {
		t.Errorf("execute bead should carry enriched description, got %q", execute.Description)
	}
	if execute.Metadata["tier"] != "advanced" {
		t.Errorf("execute bead should carry tier=advanced, got %v", execute.Metadata)
	}

	runTests1 := mc.creates[3]
	if !containsLabel(runTests1.Labels, "meow:test-run") {
		t.Errorf("run-tests-1 should carry meow:test-run label, got %v", runTests1.Labels)
	}
	if runTests1.Metadata["iteration"] != "1" {
		t.Errorf("run-tests-1 should carry iteration=1 metadata, got %v", runTests1.Metadata)
	}

	correct1 := mc.creates[4]
	if !containsLabel(correct1.Labels, "meow:test-correct") {
		t.Errorf("correct-1 should carry meow:test-correct label, got %v", correct1.Labels)
	}

	// run-tests-1 depends on execute (the only intra-sub-epic dep)
	if len(mc.deps) != 2 { // 2 test tasks × 1 run-tests→execute dep each
		t.Fatalf("expected 2 run-tests→execute deps, got %d: %v", len(mc.deps), mc.deps)
	}
	// Primary-closer mapping: taskIDs["8.1"] must be the test-task epic, not execute
	wantEpicID := "BEAD-2" // 1=section sub-epic, 2=test-task[8.1] epic
	if taskIDs["8.1"] != wantEpicID {
		t.Errorf("taskIDs[8.1]: expected test-task epic %s, got %s", wantEpicID, taskIDs["8.1"])
	}

	// Accumulators: both test-task epic IDs recorded; both execute beads recorded as leaves
	if len(p.TestTaskIDs) != 2 {
		t.Errorf("expected 2 test-task epic IDs, got %v", p.TestTaskIDs)
	}
	if len(p.LeafIDs) != 2 {
		t.Errorf("expected 2 leaf IDs (2 execute beads), got %v", p.LeafIDs)
	}
	for _, leafID := range p.LeafIDs {
		if leafID == "BEAD-2" || leafID == "BEAD-6" {
			t.Errorf("LeafIDs must contain execute beads, not test-task epics; got %v", p.LeafIDs)
		}
	}
}

func TestCollapsedTestTaskSectionGoesUnderRoot(t *testing.T) {
	mc := &mockClient{}
	p := &Planner{Client: mc, ChangeName: "test"}

	sections := []parser.Section{
		{Number: "9", Title: "Lone test", Tasks: []parser.Task{
			{Number: "9.1", Title: "Single test task"},
		}},
	}
	enriched := map[string]EnrichedTask{
		"9.1": {Task: sections[0].Tasks[0], IsTest: true, Tier: "fast"},
	}

	subEpicIDs, err := p.CreateSubEpics("ROOT", sections, enriched)
	if err != nil {
		t.Fatalf("sub-epics: %v", err)
	}

	// Collapsed: NO section sub-epic wrapper, but the test-task
	// four-bead structure still lives directly under ROOT.
	if len(mc.creates) != 4 {
		t.Fatalf("expected 4 creates (test-task epic + 3 children, no section wrapper), got %d", len(mc.creates))
	}
	if mc.creates[0].Parent != "ROOT" {
		t.Errorf("test-task epic should be a child of ROOT in the collapsed case, got parent=%q", mc.creates[0].Parent)
	}
	if !containsLabel(mc.creates[0].Labels, "meow:test") {
		t.Errorf("collapsed test-task should carry meow:test label, got %v", mc.creates[0].Labels)
	}
	// The sub-epic map points to the test-task epic (primary closer)
	if subEpicIDs[0] != "BEAD-1" {
		t.Errorf("collapsed sub-epic map should point at the test-task epic, got %q", subEpicIDs[0])
	}
}

func containsLabel(labels []string, want string) bool {
	for _, l := range labels {
		if l == want {
			return true
		}
	}
	return false
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
