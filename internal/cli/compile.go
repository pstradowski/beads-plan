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
	profileFlag   string
	dryRunFlag    bool
	parentFlag    string
	changeDirFlag string
)

var compileCmd = &cobra.Command{
	Use:   "compile <change-dir>",
	Short: "Compile OpenSpec change into apply-phase beads",
	Long: `Compile an OpenSpec change directory into a nested beads epic: sub-epics per
section, enriched leaf tasks with complexity tiers, and dependency edges from
parallelism analysis.

compile is a leaf tool. It does not orchestrate the OpenSpec lifecycle
(explore / proposal / design / verify / archive). For the full lifecycle,
pour the meow-openspec beads formula via ` + "`bd mol pour meow-openspec`" + `,
which invokes this command at its compile step.`,
	Args: cobra.ExactArgs(1),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if dryRunFlag {
			return nil // skip dep checks for dry-run
		}
		return checkBd()
	},
	RunE: runCompile,
}

func init() {
	compileCmd.Flags().StringVar(&profileFlag, "profile", "", "Provider profile for tier-to-model mapping")
	compileCmd.Flags().BoolVar(&dryRunFlag, "dry-run", false, "Preview planned structure without creating beads")
	compileCmd.Flags().StringVar(&parentFlag, "parent", "", "Parent bead ID: create the root epic as a child of this bead (for formula-step grafting)")
}

func runCompile(cmd *cobra.Command, args []string) error {
	changeDir := args[0]

	if !filepath.IsAbs(changeDir) {
		cwd, _ := os.Getwd()
		changeDir = filepath.Join(cwd, changeDir)
	}

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

	artifacts, err := parser.LoadArtifacts(changeDir)
	if err != nil {
		return fmt.Errorf("loading artifacts: %w", err)
	}

	taskTree, err := parser.ParseTasksMarkdown(artifacts.Tasks)
	if err != nil {
		return fmt.Errorf("parsing tasks: %w", err)
	}

	enriched := planner.EnrichTasks(taskTree.Sections, artifacts, nil)

	sectionParallel := planner.AnalyzeSectionParallelism(taskTree.Sections)

	var allEdges []planner.DepEdge
	allEdges = append(allEdges, sectionParallel.DepEdges...)
	for _, s := range taskTree.Sections {
		taskParallel := planner.AnalyzeTaskParallelism(s.Tasks)
		allEdges = append(allEdges, taskParallel.DepEdges...)
	}

	if dryRunFlag {
		return runCompileDryRun(artifacts, taskTree, enriched, sectionParallel, allEdges, profileName)
	}

	p := &planner.Planner{
		Client:     &planner.BdCLI{},
		ChangeName: artifacts.ChangeName,
		ParentID:   parentFlag,
	}

	rootID, err := p.CreateRootEpic()
	if err != nil {
		return fmt.Errorf("creating root epic: %w", err)
	}

	subEpicIDs, err := p.CreateSubEpics(rootID, taskTree.Sections, enriched)
	if err != nil {
		return fmt.Errorf("creating sub-epics: %w", err)
	}

	taskIDs, err := p.CreateLeafTasks(subEpicIDs, taskTree.Sections, enriched)
	if err != nil {
		return fmt.Errorf("creating leaf tasks: %w", err)
	}

	depsCreated, err := p.CreateDependencies(taskIDs, allEdges)
	if err != nil {
		return fmt.Errorf("creating dependencies: %w", err)
	}

	_ = profile // profile used indirectly via enrichment
	_ = taskIDs // task→primary-closer map, owned by the planner for dep wiring

	if jsonOutput {
		return PrintJSONCompact(buildCompileSummary(rootID, p))
	}

	fmt.Printf("Compiled epic %s with %d sections, %d tasks, %d dependencies\n",
		rootID, len(taskTree.Sections), taskTree.TotalTasks(), depsCreated)
	return nil
}

// buildCompileSummary copies the accumulators the planner filled during
// bead creation into the frozen CompileSummary shape (see
// openspec/changes/meow-openspec-execution/specs/meow-test-correction-loop/
// spec.md §"JSON summary contract"). LeafIDs contains the execute bead
// of each test-task sub-epic but not its run-tests-N or correct-N
// children; TestTaskIDs contains the test-task epic IDs.
func buildCompileSummary(rootID string, p *planner.Planner) planner.CompileSummary {
	leafIDs := append([]string{}, p.LeafIDs...)
	testTaskIDs := append([]string{}, p.TestTaskIDs...)
	tiers := make(map[string]string, len(p.Tiers))
	for k, v := range p.Tiers {
		tiers[k] = v
	}
	if leafIDs == nil {
		leafIDs = []string{}
	}
	if testTaskIDs == nil {
		testTaskIDs = []string{}
	}
	return planner.CompileSummary{
		RootID:      rootID,
		LeafIDs:     leafIDs,
		TestTaskIDs: testTaskIDs,
		Tiers:       tiers,
	}
}

func runCompileDryRun(artifacts *parser.Artifacts, tree *parser.TaskTree, enriched map[string]planner.EnrichedTask, sectionParallel planner.ParallelismResult, edges []planner.DepEdge, profileName string) error {
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
			testMarker := ""
			if e, ok := enriched[t.Number]; ok {
				tier = fmt.Sprintf(" [%s]", e.Tier)
				if e.IsTest {
					testMarker = fmt.Sprintf(" [TEST:%s]", e.TestRule)
				}
			}
			title := planner.StripTestMarker(t.Title)
			b.WriteString(fmt.Sprintf("    %s %s %s%s%s\n", status, t.Number, title, tier, testMarker))
		}
		b.WriteString("\n")
	}

	if len(edges) > 0 {
		b.WriteString(fmt.Sprintf("Dependencies: %d edges\n", len(edges)))
		for _, e := range edges {
			b.WriteString(fmt.Sprintf("  %s → %s\n", e.From, e.To))
		}
	}

	// --dry-run + --json is intentionally not a machine-readable path:
	// the CompileSummary contract is only emitted on successful bead
	// creation. Dry-run is for human preview, so print text regardless.
	fmt.Print(b.String())
	return nil
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
