package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/pstradowski/beads-plan/internal/planner"
)

func TestBuildCompileSummary(t *testing.T) {
	p := &planner.Planner{
		LeafIDs:     []string{"BEAD-11", "BEAD-12", "BEAD-21"},
		TestTaskIDs: []string{},
		Tiers: map[string]string{
			"BEAD-11": "fast",
			"BEAD-12": "standard",
			"BEAD-21": "advanced",
		},
	}

	got := buildCompileSummary("BEAD-root", p)

	if got.RootID != "BEAD-root" {
		t.Errorf("RootID: expected BEAD-root, got %q", got.RootID)
	}
	wantLeaves := []string{"BEAD-11", "BEAD-12", "BEAD-21"}
	if !reflect.DeepEqual(got.LeafIDs, wantLeaves) {
		t.Errorf("LeafIDs: expected %v, got %v", wantLeaves, got.LeafIDs)
	}
	if len(got.TestTaskIDs) != 0 {
		t.Errorf("TestTaskIDs: expected empty slice, got %v", got.TestTaskIDs)
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

func TestBuildCompileSummaryWithTestTasks(t *testing.T) {
	// Typical shape after a mixed run: three regular tasks and two
	// test tasks. LeafIDs contains execute beads for the test tasks
	// (NOT run-tests-N or correct-N), and TestTaskIDs contains the
	// two test-task epic IDs.
	p := &planner.Planner{
		LeafIDs:     []string{"LEAF-1", "TEST-EXEC-1", "LEAF-2", "TEST-EXEC-2", "LEAF-3"},
		TestTaskIDs: []string{"TEST-EPIC-1", "TEST-EPIC-2"},
		Tiers: map[string]string{
			"LEAF-1":      "fast",
			"TEST-EXEC-1": "advanced",
			"LEAF-2":      "standard",
			"TEST-EXEC-2": "standard",
			"LEAF-3":      "advanced",
		},
	}

	got := buildCompileSummary("ROOT", p)

	if len(got.LeafIDs) != 5 {
		t.Errorf("expected 5 leaf IDs (3 regular + 2 execute), got %d: %v", len(got.LeafIDs), got.LeafIDs)
	}
	for _, id := range got.LeafIDs {
		if id == "TEST-EPIC-1" || id == "TEST-EPIC-2" {
			t.Errorf("leaf_ids must not contain test-task epic IDs, got %v", got.LeafIDs)
		}
	}
	if len(got.TestTaskIDs) != 2 {
		t.Errorf("expected 2 test-task epic IDs, got %d", len(got.TestTaskIDs))
	}
	// Tiers must cover every leaf
	for _, id := range got.LeafIDs {
		if _, ok := got.Tiers[id]; !ok {
			t.Errorf("every leaf ID must have a tier entry, missing %q", id)
		}
	}
}

// TestCompileFailureEmitsNoJSONOnStdout asserts the failure-path
// scenario from meow-test-correction-loop §"JSON summary contract":
// on non-zero exit, stdout MUST be empty so the formula's plan-step
// capture doesn't ingest a half-written or error payload. This is
// an integration test that runs the binary on a non-existent change
// directory; it is skipped if the binary has not been built yet
// (run `make build` first).
func TestCompileFailureEmitsNoJSONOnStdout(t *testing.T) {
	// Walk up from the test's working directory (internal/cli) to
	// find the repo root, then look for ./build/beads-plan.
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	root := wd
	for i := 0; i < 5; i++ {
		if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
			break
		}
		root = filepath.Dir(root)
	}
	binary := filepath.Join(root, "build", "beads-plan")
	if _, err := os.Stat(binary); err != nil {
		t.Skipf("%s not built — run `make build` first", binary)
	}

	cmd := exec.Command(binary, "compile", "/nonexistent-path-for-compile-failure-test", "--json")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()

	if err == nil {
		t.Fatal("expected non-zero exit on nonexistent change dir, got success")
	}
	if stdout.Len() > 0 {
		t.Errorf("stdout must be empty on compile failure, got: %q", stdout.String())
	}
	if stderr.Len() == 0 {
		t.Error("stderr must contain diagnostics on compile failure, got empty")
	}
}

func TestBuildCompileSummaryEmptyAccumulators(t *testing.T) {
	// Fresh Planner with no beads yet — nil slices should still
	// serialize as empty arrays (not null) per the JSON contract.
	p := &planner.Planner{}

	got := buildCompileSummary("ROOT", p)

	blob, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(blob, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// leaf_ids and test_task_ids must be arrays, not null
	for _, field := range []string{"leaf_ids", "test_task_ids"} {
		if decoded[field] == nil {
			t.Errorf("%s must serialize as [], not null", field)
		}
	}
}
