package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func createTestChange(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(`## 1. Setup

- [ ] 1.1 Do setup
`), 0644)

	os.WriteFile(filepath.Join(dir, "proposal.md"), []byte("## Why\n\nTest proposal"), 0644)
	os.WriteFile(filepath.Join(dir, "design.md"), []byte("## Context\n\nTest design"), 0644)

	specDir := filepath.Join(dir, "specs", "test-cap")
	os.MkdirAll(specDir, 0755)
	os.WriteFile(filepath.Join(specDir, "spec.md"), []byte("## ADDED Requirements\n\nTest spec"), 0644)

	return dir
}

func TestLoadArtifactsFull(t *testing.T) {
	dir := createTestChange(t)

	a, err := LoadArtifacts(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if a.Tasks == "" {
		t.Error("tasks should not be empty")
	}
	if a.Proposal == "" {
		t.Error("proposal should not be empty")
	}
	if a.Design == "" {
		t.Error("design should not be empty")
	}
	if len(a.Specs) != 1 {
		t.Fatalf("expected 1 spec, got %d", len(a.Specs))
	}
	if a.Specs[0].Capability != "test-cap" {
		t.Errorf("expected capability test-cap, got %s", a.Specs[0].Capability)
	}
}

func TestLoadArtifactsMinimal(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "tasks.md"), []byte("- [ ] 1.1 Task\n"), 0644)

	a, err := LoadArtifacts(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Proposal != "" {
		t.Error("proposal should be empty for minimal change")
	}
	if a.Design != "" {
		t.Error("design should be empty for minimal change")
	}
	if len(a.Specs) != 0 {
		t.Errorf("expected 0 specs, got %d", len(a.Specs))
	}
}

func TestLoadArtifactsMissingTasks(t *testing.T) {
	dir := t.TempDir()
	// No tasks.md

	_, err := LoadArtifacts(dir)
	if err == nil {
		t.Error("expected error when tasks.md is missing")
	}
}

func TestLoadArtifactsChangeName(t *testing.T) {
	dir := createTestChange(t)

	a, err := LoadArtifacts(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.ChangeName == "" {
		t.Error("change name should not be empty")
	}
}
