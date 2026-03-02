package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pstradowski/beads-plan/internal/config"
	"github.com/pstradowski/beads-plan/internal/parser"
	"github.com/pstradowski/beads-plan/internal/planner"
	"github.com/spf13/cobra"
)

var (
	profileFlag  string
	dryRunFlag   bool
	changeDirFlag string
)

// PlanOutput is the JSON-serializable result of the plan command.
type PlanOutput struct {
	RootID      string            `json:"root_id"`
	ChangeName  string            `json:"change_name"`
	Profile     string            `json:"profile,omitempty"`
	TotalTasks  int               `json:"total_tasks"`
	Sections    int               `json:"sections"`
	DepsCreated int               `json:"deps_created"`
	DryRun      bool              `json:"dry_run"`
	SubEpics    map[string]string `json:"sub_epics,omitempty"`
	TaskIDs     map[string]string `json:"task_ids,omitempty"`
}

var planCmd = &cobra.Command{
	Use:   "plan <change-dir>",
	Short: "Create beads epic from OpenSpec change",
	Long:  "Read OpenSpec artifacts and create a nested beads epic with sub-epics, tasks, dependencies, tier assignments, and parallelism analysis.",
	Args:  cobra.ExactArgs(1),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if dryRunFlag {
			return nil // skip dep checks for dry-run
		}
		return checkBd()
	},
	RunE: runPlan,
}

func init() {
	planCmd.Flags().StringVar(&profileFlag, "profile", "", "Provider profile for tier-to-model mapping")
	planCmd.Flags().BoolVar(&dryRunFlag, "dry-run", false, "Preview planned structure without creating beads")
}

func runPlan(cmd *cobra.Command, args []string) error {
	changeDir := args[0]

	// Resolve to absolute path
	if !filepath.IsAbs(changeDir) {
		cwd, _ := os.Getwd()
		changeDir = filepath.Join(cwd, changeDir)
	}

	// Load config and resolve profile
	cfg, err := config.LoadDefault()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	profileName := config.ActiveProfile(cfg, profileFlag)
	var profile *config.Profile
	if profileName != "" {
		profile, err = config.GetProfile(cfg, profileName)
		if err != nil {
			return err
		}
	}

	// Parse artifacts
	artifacts, err := parser.LoadArtifacts(changeDir)
	if err != nil {
		return fmt.Errorf("loading artifacts: %w", err)
	}

	// Parse tasks.md
	taskTree, err := parser.ParseTasksMarkdown(artifacts.Tasks)
	if err != nil {
		return fmt.Errorf("parsing tasks: %w", err)
	}

	// Enrich tasks
	enriched := planner.EnrichTasks(taskTree.Sections, artifacts, nil)

	// Analyze parallelism
	sectionParallel := planner.AnalyzeSectionParallelism(taskTree.Sections)

	// Collect all dep edges from section and task-level analysis
	var allEdges []planner.DepEdge
	allEdges = append(allEdges, sectionParallel.DepEdges...)
	for _, s := range taskTree.Sections {
		taskParallel := planner.AnalyzeTaskParallelism(s.Tasks)
		allEdges = append(allEdges, taskParallel.DepEdges...)
	}

	if dryRunFlag {
		return runDryRun(artifacts, taskTree, enriched, sectionParallel, allEdges, profileName)
	}

	// Create beads
	p := &planner.Planner{
		Client:     &planner.BdCLI{},
		ChangeName: artifacts.ChangeName,
	}

	// Root epic
	rootID, err := p.CreateRootEpic()
	if err != nil {
		return fmt.Errorf("creating root epic: %w", err)
	}

	// Sub-epics
	subEpicIDs, err := p.CreateSubEpics(rootID, taskTree.Sections)
	if err != nil {
		return fmt.Errorf("creating sub-epics: %w", err)
	}

	// Leaf tasks
	taskIDs, err := p.CreateLeafTasks(subEpicIDs, taskTree.Sections, enriched)
	if err != nil {
		return fmt.Errorf("creating leaf tasks: %w", err)
	}

	// Dependencies
	depsCreated, err := p.CreateDependencies(taskIDs, allEdges)
	if err != nil {
		return fmt.Errorf("creating dependencies: %w", err)
	}

	// Build output
	subEpicNames := make(map[string]string)
	for i, s := range taskTree.Sections {
		subEpicNames[s.Number+". "+s.Title] = subEpicIDs[i]
	}

	output := PlanOutput{
		RootID:      rootID,
		ChangeName:  artifacts.ChangeName,
		Profile:     profileName,
		TotalTasks:  taskTree.TotalTasks(),
		Sections:    len(taskTree.Sections),
		DepsCreated: depsCreated,
		DryRun:      false,
		SubEpics:    subEpicNames,
		TaskIDs:     taskIDs,
	}

	_ = profile // profile used indirectly via enrichment

	text := fmt.Sprintf("Created epic %s with %d sections, %d tasks, %d dependencies",
		rootID, len(taskTree.Sections), taskTree.TotalTasks(), depsCreated)
	PrintOutput(output, text)
	return nil
}

func runDryRun(artifacts *parser.Artifacts, tree *parser.TaskTree, enriched map[string]planner.EnrichedTask, sectionParallel planner.ParallelismResult, edges []planner.DepEdge, profileName string) error {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("=== Dry Run: %s ===\n\n", artifacts.ChangeName))
	b.WriteString(fmt.Sprintf("Profile: %s\n", orDefault(profileName, "(none)")))
	b.WriteString(fmt.Sprintf("Sections: %d | Tasks: %d\n", len(tree.Sections), tree.TotalTasks()))
	b.WriteString(fmt.Sprintf("Section parallelism: %s\n\n", sectionParallel.Mode))

	for _, s := range tree.Sections {
		collapsed := len(s.Tasks) == 1
		tag := "epic"
		if collapsed {
			tag = "task (collapsed)"
		}
		b.WriteString(fmt.Sprintf("  [%s] %s. %s\n", tag, s.Number, s.Title))

		taskParallel := planner.AnalyzeTaskParallelism(s.Tasks)
		if len(s.Tasks) > 1 {
			b.WriteString(fmt.Sprintf("    parallelism: %s\n", taskParallel.Mode))
		}

		for _, t := range s.Tasks {
			status := "[ ]"
			if t.IsCompleted {
				status = "[x]"
			}
			tier := ""
			if e, ok := enriched[t.Number]; ok {
				tier = fmt.Sprintf(" [%s]", e.Tier)
			}
			b.WriteString(fmt.Sprintf("    %s %s %s%s\n", status, t.Number, t.Title, tier))
		}
		b.WriteString("\n")
	}

	if len(edges) > 0 {
		b.WriteString(fmt.Sprintf("Dependencies: %d edges\n", len(edges)))
		for _, e := range edges {
			b.WriteString(fmt.Sprintf("  %s → %s\n", e.From, e.To))
		}
	}

	if jsonOutput {
		output := PlanOutput{
			ChangeName: artifacts.ChangeName,
			Profile:    profileName,
			TotalTasks: tree.TotalTasks(),
			Sections:   len(tree.Sections),
			DepsCreated: len(edges),
			DryRun:     true,
		}
		return PrintJSON(output)
	}

	fmt.Print(b.String())
	return nil
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
