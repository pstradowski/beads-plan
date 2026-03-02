package viewer

import (
	"fmt"
	"strings"
	"testing"
)

// mockReader returns pre-configured BeadInfo for testing.
type mockReader struct {
	beads map[string]*BeadInfo
}

func (m *mockReader) Show(id string) (*BeadInfo, error) {
	b, ok := m.beads[id]
	if !ok {
		return nil, fmt.Errorf("bead %s not found", id)
	}
	return b, nil
}

func (m *mockReader) Children(id string) ([]string, error) {
	b, ok := m.beads[id]
	if !ok {
		return nil, fmt.Errorf("bead %s not found", id)
	}
	return b.Children, nil
}

func newMockReader() *mockReader {
	return &mockReader{
		beads: map[string]*BeadInfo{
			"ROOT": {
				ID: "ROOT", Title: "beads-plan: test-change", Type: "epic", Status: "open",
				Children: []string{"SUB-1", "SUB-2"},
			},
			"SUB-1": {
				ID: "SUB-1", Title: "1. Database Setup", Type: "epic", Status: "open",
				Children: []string{"TASK-1", "TASK-2"},
			},
			"TASK-1": {
				ID: "TASK-1", Title: "1.1 Create migration", Type: "task", Status: "closed",
				Metadata: map[string]string{"tier": "fast"},
			},
			"TASK-2": {
				ID: "TASK-2", Title: "1.2 Write handler", Type: "task", Status: "open",
				Metadata: map[string]string{"tier": "standard"},
			},
			"SUB-2": {
				ID: "SUB-2", Title: "2. API Layer", Type: "epic", Status: "open",
				Children: []string{"TASK-3"},
			},
			"TASK-3": {
				ID: "TASK-3", Title: "2.1 Implement endpoints", Type: "task", Status: "open",
				Metadata: map[string]string{"tier": "advanced"},
			},
		},
	}
}

func TestReadEpicTree(t *testing.T) {
	reader := newMockReader()

	tree, err := ReadEpicTree(reader, "ROOT")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tree.Root.ID != "ROOT" {
		t.Errorf("expected root ID=ROOT, got %s", tree.Root.ID)
	}
	if len(tree.SubEpics) != 2 {
		t.Fatalf("expected 2 sub-epics, got %d", len(tree.SubEpics))
	}
	if len(tree.SubEpics[0].Tasks) != 2 {
		t.Errorf("expected 2 tasks in sub-epic 1, got %d", len(tree.SubEpics[0].Tasks))
	}
	if len(tree.SubEpics[1].Tasks) != 1 {
		t.Errorf("expected 1 task in sub-epic 2, got %d", len(tree.SubEpics[1].Tasks))
	}
}

func TestRenderTasksMd(t *testing.T) {
	tree := &EpicTree{
		Root: BeadInfo{ID: "ROOT", Title: "test-change"},
		SubEpics: []SubEpic{
			{
				Bead: BeadInfo{ID: "SUB-1", Title: "1. Database Setup", Type: "epic", Status: "open"},
				Tasks: []BeadInfo{
					{ID: "T-1", Title: "1.1 Create migration", Status: "closed", Metadata: map[string]string{"tier": "fast"}},
					{ID: "T-2", Title: "1.2 Write handler", Status: "open", Metadata: map[string]string{"tier": "standard"}},
				},
			},
			{
				Bead: BeadInfo{ID: "SUB-2", Title: "2. API", Type: "epic", Status: "open"},
				Tasks: []BeadInfo{
					{ID: "T-3", Title: "2.1 Endpoints", Status: "open", Metadata: map[string]string{"tier": "advanced"}},
				},
			},
		},
	}

	md := RenderTasksMd(tree, DefaultRenderOptions())

	// Header comment
	if !strings.Contains(md, "<!-- Generated from beads epic ROOT") {
		t.Error("missing header comment")
	}

	// Section headers
	if !strings.Contains(md, "## 1. Database Setup") {
		t.Error("missing section 1 header")
	}
	if !strings.Contains(md, "## 2. API") {
		t.Error("missing section 2 header")
	}

	// Checkbox status
	if !strings.Contains(md, "- [x] Create migration") {
		t.Error("completed task should have [x]")
	}
	if !strings.Contains(md, "- [ ] Write handler") {
		t.Error("open task should have [ ]")
	}

	// Bead IDs and tier tags
	if !strings.Contains(md, "T-1") {
		t.Error("should contain bead ID T-1")
	}
	if !strings.Contains(md, "fast") {
		t.Error("should contain tier tag fast")
	}

	// Progress footer
	if !strings.Contains(md, "Progress: 1/3 tasks complete (33%)") {
		t.Error("missing or incorrect progress footer")
	}
}

func TestRenderTasksMdNoTags(t *testing.T) {
	tree := &EpicTree{
		Root: BeadInfo{ID: "ROOT"},
		SubEpics: []SubEpic{
			{
				Bead: BeadInfo{ID: "S1", Title: "1. Test", Type: "epic"},
				Tasks: []BeadInfo{
					{ID: "T-1", Title: "1.1 Do thing", Status: "open"},
				},
			},
		},
	}

	md := RenderTasksMd(tree, RenderOptions{ShowBeadIDs: false, ShowTiers: false})
	if strings.Contains(md, "<!--") && !strings.Contains(md, "Generated") && !strings.Contains(md, "Progress") {
		t.Error("should not have inline comments when tags disabled")
	}
}

func TestCleanTitle(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"1. Database Setup", "Database Setup"},
		{"1.1 Create migration", "Create migration"},
		{"2.3 Write handler for orders", "Write handler for orders"},
		{"No prefix here", "No prefix here"},
		{"10. Tenth Section", "Tenth Section"},
	}
	for _, tt := range tests {
		got := cleanTitle(tt.input)
		if got != tt.want {
			t.Errorf("cleanTitle(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestReadEpicTreeError(t *testing.T) {
	reader := &mockReader{beads: map[string]*BeadInfo{}}

	_, err := ReadEpicTree(reader, "MISSING")
	if err == nil {
		t.Error("expected error for missing epic")
	}
}

func TestRenderProgressAllComplete(t *testing.T) {
	tree := &EpicTree{
		Root: BeadInfo{ID: "ROOT"},
		SubEpics: []SubEpic{
			{
				Bead: BeadInfo{ID: "S1", Title: "1. Done", Type: "epic"},
				Tasks: []BeadInfo{
					{ID: "T-1", Title: "1.1 Task", Status: "closed"},
					{ID: "T-2", Title: "1.2 Task", Status: "closed"},
				},
			},
		},
	}

	md := RenderTasksMd(tree, DefaultRenderOptions())
	if !strings.Contains(md, "2/2 tasks complete (100%)") {
		t.Error("expected 100% progress")
	}
}
