package planner

import (
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
	Client    BeadClient
	ChangeName string
	Priority   string // default priority for created beads
}

// CreateRootEpic creates the top-level epic bead for the change.
func (p *Planner) CreateRootEpic() (string, error) {
	title := fmt.Sprintf("beads-plan: %s", p.ChangeName)
	priority := p.Priority
	if priority == "" {
		priority = "P1"
	}

	id, err := p.Client.Create(CreateOpts{
		Title:    title,
		Type:     "epic",
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

// CreateSubEpics creates sub-epics for each section. Sections with exactly
// one task are collapsed — the task is created directly under root instead
// of wrapping it in a sub-epic. Returns a map of section index → bead ID.
func (p *Planner) CreateSubEpics(rootID string, sections []parser.Section) (map[int]string, error) {
	ids := make(map[int]string, len(sections))

	for i, s := range sections {
		if len(s.Tasks) == 1 {
			// Collapse: create the single task directly under root
			task := s.Tasks[0]
			id, err := p.Client.Create(CreateOpts{
				Title:  fmt.Sprintf("%s %s", task.Number, task.Title),
				Type:   "task",
				Parent: rootID,
			})
			if err != nil {
				return nil, fmt.Errorf("create collapsed task for section %s: %w", s.Number, err)
			}
			ids[i] = id
		} else {
			// Multi-task section: create sub-epic
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
	}

	return ids, nil
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
}

// CreateLeafTasks creates leaf task beads under their section's sub-epic.
// For collapsed sections (single-task), the task was already created by
// CreateSubEpics, so it is skipped here. Returns map of task number → bead ID.
func (p *Planner) CreateLeafTasks(subEpicIDs map[int]string, sections []parser.Section, enriched map[string]EnrichedTask) (map[string]string, error) {
	ids := make(map[string]string)

	for i, s := range sections {
		parentID := subEpicIDs[i]

		// Skip collapsed sections — task already created as the sub-epic entry
		if len(s.Tasks) == 1 {
			ids[s.Tasks[0].Number] = parentID
			continue
		}

		for _, task := range s.Tasks {
			opts := CreateOpts{
				Title:  fmt.Sprintf("%s %s", task.Number, task.Title),
				Type:   "task",
				Parent: parentID,
			}

			// Apply enrichment if available
			if e, ok := enriched[task.Number]; ok {
				opts.Description = e.Description
				opts.Acceptance = e.Acceptance
				opts.Design = e.Design
				opts.SpecID = e.SpecID
				opts.Notes = e.Notes
				if e.Tier != "" {
					opts.Metadata = map[string]string{"tier": e.Tier}
				}
			}

			id, err := p.Client.Create(opts)
			if err != nil {
				return nil, fmt.Errorf("create leaf task %s: %w", task.Number, err)
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

// CreateDependencies creates bd dep add edges between beads.
func (p *Planner) CreateDependencies(taskIDs map[string]string, edges []DepEdge) (int, error) {
	created := 0
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
			return created, fmt.Errorf("add dep %s→%s: %w", e.From, e.To, err)
		}
		created++
	}
	return created, nil
}
