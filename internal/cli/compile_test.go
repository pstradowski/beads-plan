package cli

import (
	"reflect"
	"testing"

	"github.com/pstradowski/beads-plan/internal/parser"
	"github.com/pstradowski/beads-plan/internal/planner"
)

func TestBuildCompileSummary(t *testing.T) {
	tree := &parser.TaskTree{
		Sections: []parser.Section{
			{Number: "1", Title: "Setup", Tasks: []parser.Task{
				{Number: "1.1", Title: "First"},
				{Number: "1.2", Title: "Second"},
			}},
			{Number: "2", Title: "Impl", Tasks: []parser.Task{
				{Number: "2.1", Title: "Third"},
			}},
		},
	}
	taskIDs := map[string]string{
		"1.1": "BEAD-11",
		"1.2": "BEAD-12",
		"2.1": "BEAD-21",
	}
	enriched := map[string]planner.EnrichedTask{
		"1.1": {Tier: "fast"},
		"1.2": {Tier: "standard"},
		"2.1": {Tier: "advanced"},
	}

	got := buildCompileSummary("BEAD-root", tree, taskIDs, enriched)

	if got.RootID != "BEAD-root" {
		t.Errorf("RootID: expected BEAD-root, got %q", got.RootID)
	}
	wantLeaves := []string{"BEAD-11", "BEAD-12", "BEAD-21"}
	if !reflect.DeepEqual(got.LeafIDs, wantLeaves) {
		t.Errorf("LeafIDs: expected %v in tasks.md order, got %v", wantLeaves, got.LeafIDs)
	}
	if len(got.TestTaskIDs) != 0 {
		t.Errorf("TestTaskIDs: expected empty slice (test-task compile branch not implemented yet), got %v", got.TestTaskIDs)
	}
	wantTiers := map[string]string{
		"BEAD-11": "fast",
		"BEAD-12": "standard",
		"BEAD-21": "advanced",
	}
	if !reflect.DeepEqual(got.Tiers, wantTiers) {
		t.Errorf("Tiers: expected %v, got %v", wantTiers, got.Tiers)
	}
}

func TestBuildCompileSummarySkipsUntiered(t *testing.T) {
	tree := &parser.TaskTree{
		Sections: []parser.Section{
			{Number: "1", Title: "Setup", Tasks: []parser.Task{
				{Number: "1.1", Title: "First"},
			}},
		},
	}
	taskIDs := map[string]string{"1.1": "BEAD-11"}
	enriched := map[string]planner.EnrichedTask{
		"1.1": {Tier: ""},
	}

	got := buildCompileSummary("BEAD-root", tree, taskIDs, enriched)

	if len(got.LeafIDs) != 1 || got.LeafIDs[0] != "BEAD-11" {
		t.Errorf("LeafIDs: expected [BEAD-11], got %v", got.LeafIDs)
	}
	if _, present := got.Tiers["BEAD-11"]; present {
		t.Errorf("Tiers: untiered task must not appear in the tiers map, got %v", got.Tiers)
	}
}

func TestBuildCompileSummaryIgnoresMissingTaskIDs(t *testing.T) {
	tree := &parser.TaskTree{
		Sections: []parser.Section{
			{Number: "1", Title: "Setup", Tasks: []parser.Task{
				{Number: "1.1", Title: "First"},
				{Number: "1.2", Title: "Missing"},
			}},
		},
	}
	taskIDs := map[string]string{"1.1": "BEAD-11"}
	enriched := map[string]planner.EnrichedTask{
		"1.1": {Tier: "fast"},
	}

	got := buildCompileSummary("BEAD-root", tree, taskIDs, enriched)

	if len(got.LeafIDs) != 1 {
		t.Errorf("LeafIDs: expected only tasks with IDs, got %v", got.LeafIDs)
	}
}
