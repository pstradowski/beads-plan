package planner

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// BeadClient defines the interface for interacting with the bd CLI.
type BeadClient interface {
	Create(opts CreateOpts) (string, error)
	AddDep(issueID, dependsOnID string) error
	Close(issueID, message string) error
}

// CreateOpts holds all fields for creating a bead via bd create.
type CreateOpts struct {
	Title       string
	Type        string // "epic", "task"
	Parent      string // parent bead ID
	Priority    string // "P0", "P1", "P2"
	Description string
	Acceptance  string
	Design      string
	SpecID      string
	Notes       string
	Metadata    map[string]string
}

// bdCreateResult is the JSON output from bd create --json.
type bdCreateResult struct {
	ID string `json:"id"`
}

// BdCLI implements BeadClient by shelling out to the bd binary.
type BdCLI struct {
	// BeadsDir is the path to the .beads directory (set via BEADS_DIR env).
	BeadsDir string
}

func (b *BdCLI) cmd(args ...string) *exec.Cmd {
	c := exec.Command("bd", args...)
	if b.BeadsDir != "" {
		c.Env = append(c.Environ(), "BEADS_DIR="+b.BeadsDir)
	}
	return c
}

// Create shells out to bd create and returns the new bead ID.
func (b *BdCLI) Create(opts CreateOpts) (string, error) {
	args := []string{"create", "--json", "--silent"}

	if opts.Title != "" {
		args = append(args, "--title", opts.Title)
	}
	if opts.Type != "" {
		args = append(args, "--type", opts.Type)
	}
	if opts.Parent != "" {
		args = append(args, "--parent", opts.Parent)
	}
	if opts.Priority != "" {
		args = append(args, "--priority", opts.Priority)
	}
	if opts.Description != "" {
		args = append(args, "--description", opts.Description)
	}
	if opts.Acceptance != "" {
		args = append(args, "--acceptance", opts.Acceptance)
	}
	if opts.Design != "" {
		args = append(args, "--design", opts.Design)
	}
	if opts.SpecID != "" {
		args = append(args, "--spec-id", opts.SpecID)
	}
	if opts.Notes != "" {
		args = append(args, "--notes", opts.Notes)
	}
	for k, v := range opts.Metadata {
		args = append(args, "--metadata", k+"="+v)
	}

	cmd := b.cmd(args...)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("bd create failed: %s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", fmt.Errorf("bd create failed: %w", err)
	}

	var result bdCreateResult
	if err := json.Unmarshal(out, &result); err != nil {
		// Fallback: try to parse ID from plain text output
		id := strings.TrimSpace(string(out))
		if id != "" {
			return id, nil
		}
		return "", fmt.Errorf("failed to parse bd create output: %w", err)
	}
	return result.ID, nil
}

// AddDep shells out to bd dep add.
func (b *BdCLI) AddDep(issueID, dependsOnID string) error {
	cmd := b.cmd("dep", "add", issueID, dependsOnID)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("bd dep add %s %s failed: %s", issueID, dependsOnID, strings.TrimSpace(string(out)))
	}
	return nil
}

// Close shells out to bd close.
func (b *BdCLI) Close(issueID, message string) error {
	args := []string{"close", issueID}
	if message != "" {
		args = append(args, "-m", message)
	}
	cmd := b.cmd(args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("bd close %s failed: %s", issueID, strings.TrimSpace(string(out)))
	}
	return nil
}
