package viewer

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// BeadInfo holds the parsed output of bd show --json for a single bead.
type BeadInfo struct {
	ID       string            `json:"id"`
	Title    string            `json:"title"`
	Type     string            `json:"type"`
	Status   string            `json:"status"`
	Metadata map[string]string `json:"metadata"`
	Children []string          `json:"children"`
}

// EpicTree holds the full hierarchy read from beads.
type EpicTree struct {
	Root     BeadInfo
	SubEpics []SubEpic
}

// SubEpic holds a section-level bead and its tasks.
type SubEpic struct {
	Bead  BeadInfo
	Tasks []BeadInfo
}

// BeadReader reads bead hierarchies.
type BeadReader interface {
	Show(id string) (*BeadInfo, error)
	Children(id string) ([]string, error)
}

// BdReader implements BeadReader by shelling out to bd.
type BdReader struct {
	BeadsDir string
}

func (r *BdReader) cmd(args ...string) *exec.Cmd {
	c := exec.Command("bd", args...)
	if r.BeadsDir != "" {
		c.Env = append(c.Environ(), "BEADS_DIR="+r.BeadsDir)
	}
	return c
}

// Show returns bead info by shelling out to bd show --json.
func (r *BdReader) Show(id string) (*BeadInfo, error) {
	cmd := r.cmd("show", id, "--json")
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("bd show %s: %s", id, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, fmt.Errorf("bd show %s: %w", id, err)
	}

	// bd show --json returns a JSON array; unwrap the first element.
	var infos []BeadInfo
	if err := json.Unmarshal(out, &infos); err != nil {
		return nil, fmt.Errorf("parsing bd show output for %s: %w", id, err)
	}
	if len(infos) == 0 {
		return nil, fmt.Errorf("bd show %s: empty result", id)
	}
	infos[0].ID = id
	return &infos[0], nil
}

// Children returns the child IDs of a bead.
func (r *BdReader) Children(id string) ([]string, error) {
	info, err := r.Show(id)
	if err != nil {
		return nil, err
	}
	return info.Children, nil
}

// ReadEpicTree recursively reads an epic and all its children.
func ReadEpicTree(reader BeadReader, epicID string) (*EpicTree, error) {
	root, err := reader.Show(epicID)
	if err != nil {
		return nil, fmt.Errorf("reading root epic: %w", err)
	}

	tree := &EpicTree{Root: *root}

	for _, childID := range root.Children {
		child, err := reader.Show(childID)
		if err != nil {
			return nil, fmt.Errorf("reading child %s: %w", childID, err)
		}

		sub := SubEpic{Bead: *child}

		// If this child is an epic, read its children as tasks
		if child.Type == "epic" {
			for _, taskID := range child.Children {
				task, err := reader.Show(taskID)
				if err != nil {
					return nil, fmt.Errorf("reading task %s: %w", taskID, err)
				}
				sub.Tasks = append(sub.Tasks, *task)
			}
		}

		tree.SubEpics = append(tree.SubEpics, sub)
	}

	return tree, nil
}
