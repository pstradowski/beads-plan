package planner

import (
	"errors"
	"fmt"

	"github.com/pstradowski/beads-plan/internal/parser"
)

// PlanResult holds the IDs of all beads created during planning.
type PlanResult struct {
	RootID      string            // root epic bead ID
	SubEpicIDs  map[int]string    // section index → sub-epic bead ID (or collapsed task ID)
	LeafTaskIDs map[string]string // task number (e.g., "1.1") → bead ID
	DepsCreated int               // number of dependency edges created
}

// Planner orchestrates bead creation from parsed OpenSpec artifacts.
type Planner struct {
	Client     BeadClient
	ChangeName string
	Priority   string // default priority for created beads
	ParentID   string // optional parent bead; when set, the root epic becomes a child of this bead

	// Accumulators populated as beads are created. These feed the
	// CompileSummary emitted on --json. Consumers should not mutate.
	LeafIDs     []string          // leaf bead IDs in tasks.md order (execute beads for tests, task beads for regulars). Excludes run-tests-N and correct-N.
	TestTaskIDs []string          // test-task epic bead IDs in tasks.md order
	Tiers       map[string]string // leaf bead ID → tier string
}

func (p *Planner) recordLeaf(id, tier string) {
	p.LeafIDs = append(p.LeafIDs, id)
	if tier != "" {
		if p.Tiers == nil {
			p.Tiers = map[string]string{}
		}
		p.Tiers[id] = tier
	}
}

// CreateRootEpic creates the top-level epic bead for the change. When
// ParentID is set on the Planner, the root is created as a child of that
// bead so that an external orchestrator (e.g. a beads formula step) can
// graft the compile output into an existing molecule.
func (p *Planner) CreateRootEpic() (string, error) {
	title := fmt.Sprintf("beads-plan: %s", p.ChangeName)
	priority := p.Priority
	if priority == "" {
		priority = "P1"
	}

	id, err := p.Client.Create(CreateOpts{
		Title:    title,
		Type:     "epic",
		Parent:   p.ParentID,
		Priority: priority,
		Metadata: map[string]string{
			"change": p.ChangeName,
		},
	})
	if err != nil {
		return "", fmt.Errorf("create root epic: %w", err)
	}
	return id, nil
}

// CompileSummary is the machine-readable result of a compile run, emitted
// on stdout when `--json` is passed. It is the contract the meow-openspec
// formula's plan step consumes to graft compiled atoms into its molecule.
//
// Shape is frozen by openspec/changes/meow-openspec-execution/specs/
// meow-test-correction-loop/spec.md §"JSON summary contract". No other
// top-level fields are emitted.
type CompileSummary struct {
	RootID      string            `json:"root_id"`
	LeafIDs     []string          `json:"leaf_ids"`
	TestTaskIDs []string          `json:"test_task_ids"`
	Tiers       map[string]string `json:"tiers"`
}

// CreateSubEpics creates sub-epics for each section. Sections with exactly
// one task are collapsed — the task (or its test-task sub-epic) is created
// directly under root instead of wrapping it in a section sub-epic.
// Returns a map of section index → bead ID (the sub-epic for multi-task
// sections, or the primary-closer ID for collapsed sections: the test-task
// epic for test tasks, or the leaf bead for regulars).
func (p *Planner) CreateSubEpics(rootID string, sections []parser.Section, enriched map[string]EnrichedTask) (map[int]string, error) {
	ids := make(map[int]string, len(sections))

	for i, s := range sections {
		if len(s.Tasks) == 1 {
			// Collapsed: create the single task directly under root.
			task := s.Tasks[0]
			e := enriched[task.Number]
			id, err := p.createTaskAtoms(rootID, task, e)
			if err != nil {
				return nil, fmt.Errorf("create collapsed task for section %s: %w", s.Number, err)
			}
			ids[i] = id
			continue
		}

		// Multi-task section: create sub-epic wrapper.
		title := fmt.Sprintf("%s. %s", s.Number, s.Title)
		id, err := p.Client.Create(CreateOpts{
			Title:  title,
			Type:   "epic",
			Parent: rootID,
		})
		if err != nil {
			return nil, fmt.Errorf("create sub-epic for section %s: %w", s.Number, err)
		}
		ids[i] = id
	}

	return ids, nil
}

// createTaskAtoms creates the bead structure for a single task and returns
// the "primary closer" bead ID — the ID that downstream deps should wait
// for. For regular tasks this is the leaf bead. For test tasks this is
// the run-tests-1 bead (type=task), which satisfies bd's constraint that
// deps must be between tasks. Semantically, downstream tasks wait for the
// test run to pass. (GH#3)
func (p *Planner) createTaskAtoms(parent string, task parser.Task, e EnrichedTask) (string, error) {
	if e.IsTest {
		return p.createTestTaskSubEpic(parent, task, e)
	}
	return p.createRegularLeaf(parent, task, e)
}

func (p *Planner) createRegularLeaf(parent string, task parser.Task, e EnrichedTask) (string, error) {
	opts := CreateOpts{
		Title:       fmt.Sprintf("%s %s", task.Number, task.Title),
		Type:        "task",
		Parent:      parent,
		Description: e.Description,
		Acceptance:  e.Acceptance,
		Design:      e.Design,
		SpecID:      e.SpecID,
		Notes:       e.Notes,
	}
	if e.Tier != "" {
		opts.Metadata = map[string]string{"tier": e.Tier}
	}
	id, err := p.Client.Create(opts)
	if err != nil {
		return "", fmt.Errorf("create leaf task %s: %w", task.Number, err)
	}
	p.recordLeaf(id, e.Tier)
	return id, nil
}

// createTestTaskSubEpic compiles a detected test task into the four-bead
// structure mandated by the meow-test-correction-loop spec:
//
//	test-task (epic, labeled meow:test)
//	├── execute        (carries the enriched description)
//	├── run-tests-1    (labeled meow:test-run, depends on execute)
//	└── correct-1      (labeled meow:test-correct, empty stub)
//
// The run-tests-N → correct-N retry loop itself is driven by the
// meow-openspec formula at runtime, not here at compile time.
//
// Returns the run-tests-1 bead ID as the "primary closer" for dependency
// wiring — this is a type=task bead that satisfies bd's constraint that
// deps must be between tasks (not task→epic). Downstream tasks that
// depend on this test task will wait for run-tests-1 to close, which
// semantically means "the tests passed." (GH#3)
//
// The execute bead ID is appended to p.LeafIDs so it shows up in
// CompileSummary.LeafIDs (run-tests-1 and correct-1 do not).
func (p *Planner) createTestTaskSubEpic(parent string, task parser.Task, e EnrichedTask) (string, error) {
	cleanTitle := StripTestMarker(task.Title)

	epicMeta := map[string]string{}
	if e.TestRule != "" {
		epicMeta["meow_detect_rule"] = e.TestRule
	}
	epicID, err := p.Client.Create(CreateOpts{
		Title:    fmt.Sprintf("%s %s", task.Number, cleanTitle),
		Type:     "epic",
		Parent:   parent,
		Labels:   []string{"meow:test"},
		Metadata: epicMeta,
	})
	if err != nil {
		return "", fmt.Errorf("create test-task epic %s: %w", task.Number, err)
	}
	p.TestTaskIDs = append(p.TestTaskIDs, epicID)

	executeOpts := CreateOpts{
		Title:       fmt.Sprintf("%s execute: %s", task.Number, cleanTitle),
		Type:        "task",
		Parent:      epicID,
		Description: e.Description,
		Acceptance:  e.Acceptance,
		Design:      e.Design,
		SpecID:      e.SpecID,
		Notes:       e.Notes,
	}
	if e.Tier != "" {
		executeOpts.Metadata = map[string]string{"tier": e.Tier}
	}
	executeID, err := p.Client.Create(executeOpts)
	if err != nil {
		return "", fmt.Errorf("create test-task execute %s: %w", task.Number, err)
	}
	p.recordLeaf(executeID, e.Tier)

	runID, err := p.Client.Create(CreateOpts{
		Title:    fmt.Sprintf("%s run-tests-1", task.Number),
		Type:     "task",
		Parent:   epicID,
		Labels:   []string{"meow:test-run"},
		Metadata: map[string]string{"iteration": "1"},
	})
	if err != nil {
		return "", fmt.Errorf("create test-task run-tests-1 %s: %w", task.Number, err)
	}
	if err := p.Client.AddDep(runID, executeID); err != nil {
		return "", fmt.Errorf("wire run-tests-1 → execute for %s: %w", task.Number, err)
	}

	if _, err := p.Client.Create(CreateOpts{
		Title:    fmt.Sprintf("%s correct-1", task.Number),
		Type:     "task",
		Parent:   epicID,
		Labels:   []string{"meow:test-correct"},
		Metadata: map[string]string{"iteration": "1"},
	}); err != nil {
		return "", fmt.Errorf("create test-task correct-1 %s: %w", task.Number, err)
	}

	return runID, nil
}

// EnrichedTask extends a parsed task with additional context for bead creation.
type EnrichedTask struct {
	parser.Task
	Description string
	Acceptance  string
	Design      string
	SpecID      string
	Notes       string
	Tier        string // "fast", "standard", "advanced"
	IsTest      bool   // task is a test task — compiles into a four-bead sub-epic
	TestRule    string // which detection rule fired: "explicit" | "section" | "keyword" | "" (none)
}

// CreateLeafTasks creates leaf task beads (or test-task sub-epics) under
// their section's sub-epic. For collapsed single-task sections, the task
// structure was already created by CreateSubEpics under rootID, so this
// method reuses that ID without re-creating. Returns map of task number
// → primary-closer bead ID (regular leaf for non-test tasks, run-tests-1
// for test tasks) — that ID is what CreateDependencies wires as the
// dep source/target for a given task number.
func (p *Planner) CreateLeafTasks(subEpicIDs map[int]string, sections []parser.Section, enriched map[string]EnrichedTask) (map[string]string, error) {
	ids := make(map[string]string)

	for i, s := range sections {
		parentID := subEpicIDs[i]

		// Collapsed single-task section: already created by CreateSubEpics.
		if len(s.Tasks) == 1 {
			ids[s.Tasks[0].Number] = parentID
			continue
		}

		for _, task := range s.Tasks {
			e := enriched[task.Number]
			id, err := p.createTaskAtoms(parentID, task, e)
			if err != nil {
				return nil, err
			}
			ids[task.Number] = id
		}
	}

	return ids, nil
}

// DepEdge represents a dependency between two tasks by their task number.
type DepEdge struct {
	From string // task number that depends on...
	To   string // ...this task number
}

// CreateDependencies creates bd dep add edges between beads. Individual
// edge failures are collected and reported as a combined error after all
// edges have been attempted, so that a single bad edge (e.g., a type
// constraint violation) does not discard all remaining valid edges. (GH#4)
func (p *Planner) CreateDependencies(taskIDs map[string]string, edges []DepEdge) (int, error) {
	created := 0
	var errs []error
	for _, e := range edges {
		fromID, ok := taskIDs[e.From]
		if !ok {
			continue // skip edges for unknown tasks
		}
		toID, ok := taskIDs[e.To]
		if !ok {
			continue
		}
		if err := p.Client.AddDep(fromID, toID); err != nil {
			errs = append(errs, fmt.Errorf("add dep %s→%s: %w", e.From, e.To, err))
			continue
		}
		created++
	}
	if len(errs) > 0 {
		return created, fmt.Errorf("created %d deps, %d failed: %w", created, len(errs), errors.Join(errs...))
	}
	return created, nil
}
