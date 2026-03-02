package planner

import (
	"strings"

	"github.com/pstradowski/beads-plan/internal/parser"
)

// ParallelismMode describes whether items can run concurrently.
type ParallelismMode string

const (
	ModeParallel   ParallelismMode = "parallel"
	ModeSequential ParallelismMode = "sequential"
	ModeMixed      ParallelismMode = "mixed"
)

// ParallelismResult holds the analysis output for a group of items.
type ParallelismResult struct {
	Mode           ParallelismMode
	ParallelGroups [][]string // groups of task/section numbers that can run together
	DepEdges       []DepEdge  // dependency edges discovered
}

// AnalyzeSectionParallelism determines which sections can run concurrently.
// It detects cross-section references by checking if task titles in section B
// mention keywords from section A's title or tasks.
func AnalyzeSectionParallelism(sections []parser.Section) ParallelismResult {
	n := len(sections)
	if n <= 1 {
		groups := make([][]string, 0, n)
		for _, s := range sections {
			groups = append(groups, []string{s.Number})
		}
		return ParallelismResult{
			Mode:           ModeSequential,
			ParallelGroups: groups,
		}
	}

	// Build keyword sets per section from titles and task titles
	sectionKeywords := make([][]string, n)
	for i, s := range sections {
		kws := extractKeywords(s.Title)
		for _, t := range s.Tasks {
			kws = append(kws, extractKeywords(t.Title)...)
		}
		sectionKeywords[i] = kws
	}

	// Detect cross-section dependencies: if section j mentions section i's keywords
	// and i < j, then j depends on i
	deps := map[int][]int{} // j → [i, ...] (j depends on i)
	var edges []DepEdge
	for j := 1; j < n; j++ {
		jText := strings.ToLower(sections[j].Title)
		for _, t := range sections[j].Tasks {
			jText += " " + strings.ToLower(t.Title)
		}
		for i := 0; i < j; i++ {
			if hasCrossReference(jText, sectionKeywords[i]) {
				deps[j] = append(deps[j], i)
				edges = append(edges, DepEdge{
					From: sections[j].Number,
					To:   sections[i].Number,
				})
			}
		}
	}

	// Build parallel groups using topological waves
	groups := buildWaves(n, deps)
	mode := classifyMode(groups)

	// Convert groups from indices to section numbers
	namedGroups := make([][]string, len(groups))
	for gi, group := range groups {
		namedGroups[gi] = make([]string, len(group))
		for ti, idx := range group {
			namedGroups[gi][ti] = sections[idx].Number
		}
	}

	return ParallelismResult{
		Mode:           mode,
		ParallelGroups: namedGroups,
		DepEdges:       edges,
	}
}

// AnalyzeTaskParallelism determines which tasks within a section can run concurrently.
func AnalyzeTaskParallelism(tasks []parser.Task) ParallelismResult {
	n := len(tasks)
	if n <= 1 {
		groups := make([][]string, 0, n)
		for _, t := range tasks {
			groups = append(groups, []string{t.Number})
		}
		return ParallelismResult{
			Mode:           ModeSequential,
			ParallelGroups: groups,
		}
	}

	// Detect intra-section dependencies via keyword overlap
	taskKeywords := make([][]string, n)
	for i, t := range tasks {
		taskKeywords[i] = extractKeywords(t.Title)
	}

	deps := map[int][]int{}
	var edges []DepEdge
	for j := 1; j < n; j++ {
		jText := strings.ToLower(tasks[j].Title)
		for i := 0; i < j; i++ {
			if hasCrossReference(jText, taskKeywords[i]) {
				deps[j] = append(deps[j], i)
				edges = append(edges, DepEdge{
					From: tasks[j].Number,
					To:   tasks[i].Number,
				})
			}
		}
	}

	groups := buildWaves(n, deps)
	mode := classifyMode(groups)

	namedGroups := make([][]string, len(groups))
	for gi, group := range groups {
		namedGroups[gi] = make([]string, len(group))
		for ti, idx := range group {
			namedGroups[gi][ti] = tasks[idx].Number
		}
	}

	return ParallelismResult{
		Mode:           mode,
		ParallelGroups: namedGroups,
		DepEdges:       edges,
	}
}

// extractKeywords returns meaningful lowercase keywords from text.
// Filters out common short/stop words.
func extractKeywords(text string) []string {
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"for": true, "to": true, "in": true, "of": true, "with": true,
		"is": true, "it": true, "on": true, "at": true, "by": true,
		"from": true, "as": true, "be": true, "this": true, "that": true,
		"add": true, "create": true, "implement": true, "write": true,
		"update": true, "set": true, "up": true, "new": true,
	}

	words := strings.Fields(strings.ToLower(text))
	var keywords []string
	for _, w := range words {
		w = strings.Trim(w, ".,;:!?()[]{}\"'")
		if len(w) >= 3 && !stopWords[w] {
			keywords = append(keywords, w)
		}
	}
	return keywords
}

// hasCrossReference checks if text mentions any of the given keywords.
// Requires at least one meaningful keyword match.
func hasCrossReference(text string, keywords []string) bool {
	for _, kw := range keywords {
		if len(kw) >= 4 && strings.Contains(text, kw) {
			return true
		}
	}
	return false
}

// buildWaves groups indices into execution waves using topological sorting.
// Items with no unsatisfied dependencies go in the earliest possible wave.
func buildWaves(n int, deps map[int][]int) [][]int {
	// Compute in-degree
	inDeg := make([]int, n)
	for _, ds := range deps {
		for range ds {
			// Count deps per dependent
		}
	}
	for j, ds := range deps {
		_ = j
		inDeg[j] = len(ds)
	}

	placed := make([]bool, n)
	var waves [][]int

	for {
		var wave []int
		for i := 0; i < n; i++ {
			if placed[i] {
				continue
			}
			// Check if all dependencies are satisfied
			allSatisfied := true
			if ds, ok := deps[i]; ok {
				for _, d := range ds {
					if !placed[d] {
						allSatisfied = false
						break
					}
				}
			}
			if allSatisfied {
				wave = append(wave, i)
			}
		}
		if len(wave) == 0 {
			// Remaining items have circular deps — force them into a wave
			for i := 0; i < n; i++ {
				if !placed[i] {
					wave = append(wave, i)
				}
			}
			if len(wave) == 0 {
				break
			}
		}
		for _, i := range wave {
			placed[i] = true
		}
		waves = append(waves, wave)

		// Check if all placed
		allPlaced := true
		for _, p := range placed {
			if !p {
				allPlaced = false
				break
			}
		}
		if allPlaced {
			break
		}
	}

	return waves
}

// classifyMode determines the parallelism mode from the wave structure.
func classifyMode(groups [][]int) ParallelismMode {
	if len(groups) <= 1 {
		if len(groups) == 1 && len(groups[0]) > 1 {
			return ModeParallel
		}
		return ModeSequential
	}
	hasParallel := false
	hasSequential := false
	for _, g := range groups {
		if len(g) > 1 {
			hasParallel = true
		}
		if len(g) == 1 {
			hasSequential = true
		}
	}
	if hasParallel && hasSequential {
		return ModeMixed
	}
	if hasParallel {
		return ModeParallel
	}
	return ModeSequential
}
