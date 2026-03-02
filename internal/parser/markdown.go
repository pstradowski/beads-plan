package parser

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"
)

// Task represents a single checkbox item from tasks.md.
type Task struct {
	Number      string // e.g., "1.1", "2.3"
	Title       string // task description text
	IsCompleted bool   // true if [x]
}

// Section represents an H2 group of tasks.
type Section struct {
	Number string // e.g., "1", "2"
	Title  string // section title without number
	Tasks  []Task
}

// TaskTree is the parsed result of a tasks.md file.
type TaskTree struct {
	Sections []Section
}

var (
	sectionRe  = regexp.MustCompile(`^##\s+(\d+)\.\s+(.+)$`)
	checkboxRe = regexp.MustCompile(`^-\s+\[([ xX])\]\s+(\d+(?:\.\d+)*)\s+(.+)$`)
)

// ParseTasksMarkdown parses a tasks.md string into a TaskTree.
func ParseTasksMarkdown(content string) (*TaskTree, error) {
	tree := &TaskTree{}
	var currentSection *Section

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Try section header
		if m := sectionRe.FindStringSubmatch(line); m != nil {
			if currentSection != nil {
				tree.Sections = append(tree.Sections, *currentSection)
			}
			currentSection = &Section{
				Number: m[1],
				Title:  strings.TrimSpace(m[2]),
			}
			continue
		}

		// Try checkbox item
		if m := checkboxRe.FindStringSubmatch(line); m != nil {
			checked := m[1] == "x" || m[1] == "X"
			task := Task{
				Number:      m[2],
				Title:       strings.TrimSpace(m[3]),
				IsCompleted: checked,
			}
			if currentSection == nil {
				// Flat file with no sections — create default
				currentSection = &Section{
					Number: "1",
					Title:  "Tasks",
				}
			}
			currentSection.Tasks = append(currentSection.Tasks, task)
			continue
		}
		// Skip non-matching lines (blank, comments, etc.)
	}

	if currentSection != nil {
		tree.Sections = append(tree.Sections, *currentSection)
	}

	if len(tree.Sections) == 0 || tree.TotalTasks() == 0 {
		return nil, fmt.Errorf("no tasks found in input")
	}

	return tree, nil
}

// TotalTasks returns the total number of tasks across all sections.
func (t *TaskTree) TotalTasks() int {
	count := 0
	for _, s := range t.Sections {
		count += len(s.Tasks)
	}
	return count
}

// CompletedTasks returns the number of completed tasks.
func (t *TaskTree) CompletedTasks() int {
	count := 0
	for _, s := range t.Sections {
		for _, task := range s.Tasks {
			if task.IsCompleted {
				count++
			}
		}
	}
	return count
}
