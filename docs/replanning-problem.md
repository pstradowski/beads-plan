# The Replanning Problem in Multi-Agent Task Execution

## Problem Statement

In a multi-agent system where work is decomposed into a dependency graph of tasks (beads), a completed task's output may reveal information that invalidates the assumptions behind downstream tasks. The plan was correct *at planning time*, but execution produces knowledge that requires the remaining plan to change.

Examples:

- **Discovery**: A task to "add a new API endpoint" discovers the existing router architecture can't support the needed pattern. Downstream tasks assumed the endpoint exists.
- **Decision**: A task to "evaluate storage options" concludes that Redis is needed instead of the planned PostgreSQL approach. All downstream tasks reference PostgreSQL schemas.
- **Scope change**: A task to "implement auth middleware" reveals the auth library doesn't support the assumed token format. Related tasks need different integration patterns.
- **Elimination**: A task's output makes a downstream task unnecessary (e.g., a library already provides functionality that was planned to be built from scratch).
- **Spawning**: A task reveals new work that wasn't anticipated — new subtasks that must be completed before the original downstream tasks can proceed.

The core tension: **plans are static artifacts, but execution is dynamic**. The more detailed and pre-structured the plan, the more brittle it becomes when assumptions break.

## Current State

### beads-plan (TradeBase)

beads-plan is a one-shot compiler: OpenSpec `tasks.md` → bead molecule. It produces the plan but explicitly does not run it.

**What exists toward replanning:**

1. **Task output protocol** — Every leaf bead is enriched with instructions to capture structured output:
   ```
   files_changed: [list of paths]
   decisions: [architectural/implementation decisions]
   discoveries: [unexpected findings or issues]
   ```
   This creates the *data channel* for replan triggers, but nothing reads or acts on it.

2. **Wave-based parallelism** — Tasks are grouped into execution waves via `parallel_groups` metadata. An orchestrator *could* pause between waves to evaluate outputs before launching the next wave. But no such orchestrator exists.

3. **`beads-plan view`** — Regenerates `tasks.md` from live bead state. This is a read operation for humans or the verify step, not a replan trigger.

4. **Dependency edges** (`bd dep add`) — Authoritative constraints that `bd ready` respects. If a task's dependencies aren't met, it stays blocked. But the system can't *create new* dependencies based on runtime discoveries.

**What's missing:**

- No automatic detection of "this output invalidates that assumption"
- No incremental/diff-aware replanning (only full replan from scratch)
- No feedback loop from completed tasks back to the planning layer
- No orchestrator to coordinate between waves

### Gas Town (Steve Yegge)

Gas Town is a multi-agent workspace manager that uses the same beads infrastructure with a MEOW (Molecular Expression of Work) stack.

**What exists toward replanning:**

1. **The Refinery agent** — A merge-management agent that handles conflicting merges. When "so much has changed that the original work doesn't even make sense anymore," the Refinery can "creatively re-imagine" the changes to fit the new codebase reality. This is reactive adaptation at merge time, not proactive replanning.

2. **Molecules with gates and loops** — Workflow templates (protomolecules) can include approval gates and conditional branching (e.g., failed review → loop back to implementation). This handles *expected* replan scenarios baked into the template. It does not handle unexpected discoveries.

3. **Witness agent + heartbeat nudging** — Supervisor agents periodically poke workers to detect stalls. This catches stuck agents but doesn't detect "this agent's output means the next 5 tasks are wrong."

4. **Crash durability** — Workflows survive agent crashes, context exhaustion, and session restarts via persistent state in git-backed beads. "Keep throwing agents at the work until it eventually finishes." But surviving a crash is not the same as adapting a plan.

5. **Human steering** — Gas Town explicitly requires constant human oversight. Multiple analyses describe it as not a hands-off system.

**What's missing (same gaps as beads-plan):**

- No structured output protocol for capturing discoveries (beads-plan is actually ahead here)
- No automatic detection of assumption violations
- No incremental replanning mechanism
- The Refinery solves a narrower problem (merge conflicts), not plan invalidation

## Why This Is Hard

### 1. Detection is an open problem

How does a system know that "we chose Redis" invalidates downstream PostgreSQL tasks? This requires either:
- **Semantic understanding** of task descriptions and how outputs relate to assumptions (LLM-grade reasoning)
- **Explicit assumption tracking** where each task declares its assumptions and each output is checked against them
- **Human judgment** (what both systems currently rely on)

### 2. Replanning scope is ambiguous

When a discovery invalidates part of the plan:
- Which downstream tasks are affected? (could be 1, could be all)
- Should affected tasks be updated, replaced, or deleted?
- Do new tasks need to be created?
- Do dependency edges need rewiring?
- Does the change affect the OpenSpec design/specs too, or just the tasks?

### 3. Incremental vs. full replan

Full replan (`beads-plan compile` from scratch) is simpler but wasteful — it discards completed work context and creates duplicate beads. Incremental replan requires diffing the old plan against the new one and surgically updating only what changed, which is significantly more complex.

### 4. Orchestrator authority

Who decides to replan? Options:
- **The completing agent** flags a discovery → someone else decides
- **A supervisor agent** reviews outputs between waves → decides to replan
- **The human** notices and intervenes (current state of both systems)
- **An automated rule** triggers replan when certain output patterns are detected

## Potential Approaches

### Approach A: Wave-gate orchestrator (incremental, practical)

Add an orchestrator layer that pauses between execution waves:

```
Wave 1 executes → all tasks complete
  ↓
Orchestrator reads task outputs (files_changed, decisions, discoveries)
  ↓
If discoveries contain replan triggers:
  → Flag affected downstream tasks
  → Present to human/supervisor for replan decision
  → Either: update affected task descriptions in-place
  → Or: run beads-plan compile again for remaining work
  ↓
Wave 2 executes (with updated tasks)
```

**Pros**: Leverages existing wave structure and task output protocol. Bounded blast radius (only affects next wave).
**Cons**: Only catches issues at wave boundaries. Within-wave discoveries wait until wave completes.

### Approach B: Assumption tracking (structured, ambitious)

Each task explicitly declares its assumptions:

```yaml
task: "Implement Redis caching layer"
assumes:
  - storage_backend: redis        # from task "Evaluate storage options"
  - auth_token_format: jwt        # from task "Design auth flow"
outputs:
  - storage_config_path: string
  - cache_ttl_strategy: string
```

When a task completes, its outputs are checked against downstream assumptions. Violated assumptions trigger targeted replanning of affected tasks only.

**Pros**: Precise detection. Surgical replanning. Self-documenting dependency semantics.
**Cons**: High overhead at planning time. Assumption schemas are hard to define generically. Requires LLM assistance to extract assumptions from natural-language task descriptions.

### Approach C: Supervisor-driven review (LLM-powered, pragmatic)

A supervisor agent (like Gas Town's Witness, but smarter) reviews each completed task's output:

```
Task completes → supervisor reads output
  ↓
Supervisor LLM prompt:
  "Given this task's output and the remaining plan,
   do any downstream tasks need to change?
   If yes, which ones and how?"
  ↓
If changes needed:
  → Update task descriptions via bd update
  → Add/remove dependency edges
  → Flag for human review if confidence is low
```

**Pros**: Uses LLM reasoning to bridge the semantic gap. No schema overhead. Works with natural-language tasks.
**Cons**: LLM judgment is unreliable for subtle assumption violations. Expensive (every task completion triggers a review). Supervisor context window fills up on large plans.

### Approach D: Re-derive and diff (full replan with reconciliation)

After significant tasks complete, re-run the full planning process against updated context, then diff against the existing bead molecule:

```
Tasks complete → update OpenSpec design/specs with new knowledge
  ↓
beads-plan compile --diff <change-dir> <existing-epic-id>
  ↓
Diff output:
  - Tasks to keep (unchanged)
  - Tasks to update (description/deps changed)
  - Tasks to delete (no longer needed)
  - Tasks to add (new work discovered)
  ↓
Apply diff (with human approval for destructive changes)
```

**Pros**: Leverages existing planning infrastructure. Clean separation between "what changed" and "what to do about it."
**Cons**: Requires updating OpenSpec artifacts first (manual step). Diff algorithm for task graphs is non-trivial. Risk of ID/reference breakage.

## Recommendation for beads-plan

Start with **Approach A** (wave-gate orchestrator) because:

1. The building blocks already exist: wave structure (`parallel_groups`), task output protocol (`discoveries`), and `bd ready` for gating.
2. It's the smallest useful increment — doesn't require changing the planning format or adding assumption schemas.
3. It degrades gracefully — if the orchestrator misses something, the human can still intervene (current behavior).
4. It provides concrete data (discovery logs) to inform whether more sophisticated approaches (B, C, D) are worth building.

The minimum viable implementation:
1. An orchestrator script that runs `bd ready`, dispatches a wave, waits for completion.
2. After each wave, it reads completed tasks' `discoveries` metadata.
3. If any discoveries are non-empty, it pauses and surfaces them for review (initially to the human, later to a supervisor LLM).
4. The human/supervisor decides: continue as planned, update specific tasks, or trigger full replan.

This keeps beads-plan as a compiler (its current strength) and adds the orchestrator as a separate, thin layer on top.

## Open Questions

1. **Granularity of discovery classification**: Should the output protocol distinguish between informational discoveries (no replan needed) and blocking discoveries (replan required)? E.g., `discoveries.info` vs `discoveries.blocking`.

2. **Replan authority**: Should replanning require human approval, or can a supervisor agent be trusted to modify the plan autonomously? The risk of autonomous replanning is plan drift — the final result may diverge significantly from the original design.

3. **OpenSpec coherence**: If tasks change significantly during execution, the OpenSpec artifacts (design, specs) become stale. Should replanning also update the specs, or is that a separate concern?

4. **Cost model**: Each replan evaluation (especially LLM-powered) has a token cost. How do we avoid the orchestrator spending more on meta-reasoning than the agents spend on actual work?

5. **Convergence**: How do we prevent infinite replan loops where each replan triggers discoveries that trigger more replans? A maximum replan depth or a "good enough" threshold?

## References

- [Gas Town GitHub](https://github.com/steveyegge/gastown)
- [Gas Town's Agent Patterns (Maggie Appleton)](https://maggieappleton.com/gastown)
- [GasTown and the Two Kinds of Multi-Agent](https://paddo.dev/blog/gastown-two-kinds-of-multi-agent/)
- [Gas Town Docs](https://docs.gastownhall.ai/)
- beads-plan design: `openspec/changes/archive/2026-03-03-beads-tasks-view/design.md`
- beads-plan task output protocol: `beads-plan/internal/planner/enrichment.go`
