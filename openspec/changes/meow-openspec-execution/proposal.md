## Why

Today `beads-plan plan` compiles a single OpenSpec artifact — `tasks.md` — into a bead molecule (this change renames it to `compile`; see §What Changes). That covers only the **apply** phase of the OpenSpec lifecycle. Everything around it — exploring the problem, drafting the proposal, writing specs and design, verifying the implementation, and archiving — runs as ad-hoc skill invocations with no shared workflow state. The operator has to remember where they are, what's been approved, and when to pause for review.

This creates three concrete problems:

1. **No template.** Every new OpenSpec change re-derives the same lifecycle from scratch. There is no reusable MEOW formula (`Formula → Protomolecule → Molecule → Epics → Beads`) that names the phases, their order, or their gate conditions. beads-plan sits at the Protomolecule→Molecule boundary but only for one phase of the lifecycle.
2. **Human-in-the-loop is implicit.** The vanilla OpenSpec workflow has decision points (accept proposal, accept design, accept verify, archive), but nothing forces the operator to stop. An autonomous agent can (and will) plow through the whole lifecycle, including archiving a change that was never reviewed. We need explicit gate beads at phase transitions that cannot be bypassed.
3. **Test failures have no correction contract.** When a task in tasks.md is a test task and the tests fail, the fix is left to whatever the agent happens to do next. There is no structured `execute → run → (on-fail) correct → re-run` sub-molecule, so a failed test can silently turn into a ticked checkbox.

Beads already has the primitives we need: `bd formula` (reusable templates), `bd cook`/`bd mol pour` (instantiate a formula into a persistent molecule), `bd gate` (async wait conditions — `human`, `timer`, `gh:run`, `gh:pr`, `bead`), and `bd mol ready --gated` (discovery-based gate-resume). We just have not assembled them into an OpenSpec-shaped workflow.

**Architectural stance: beads drives, beads-plan is a tool.** The operator's entry point is `bd mol pour meow-openspec` — not a new beads-plan subcommand. beads-plan is invoked *by* a formula step to compile apply-phase atoms, the same way any other shell command could be invoked by a step. All workflow semantics (phases, gates, retry loops, gate-resume) live in the formula where `bd mol show`, `bd mol current`, and `bd mol ready --gated` can see and manipulate them. Nothing is hidden inside a Go binary.

## What Changes

1. **New beads formula: `meow-openspec`.** A `.beads/formulas/meow-openspec.formula.toml` file that models the OpenSpec lifecycle as a DAG of template steps: `explore → proposal → specs → design → tasks → plan → apply → verify → archive`. Phase-transition steps have `gate: { type: "human" }` attached. **Four gates are mandatory** and cannot be suppressed: `proposal-accepted`, `design-accepted`, `verify-accepted`, `archive-accepted`. Two more (`specs-accepted`, `tasks-accepted`) are on by default but can be suppressed per change via `--skip-gates`. An optional `explore-accepted` gate is off by default.

2. **Formula phase: liquid.** Instantiated via `bd mol pour` (not `wisp`). OpenSpec changes have audit value — operators will want to look back at when proposal was accepted, what the verify report said, and which gates were resolved by whom. The molecule syncs via `bd dolt push` alongside the rest of the beads store.

3. **Formula step invokes beads-plan as a subprocess.** Step 6 (`plan`) runs `beads-plan compile {{change_dir}} --parent {{step.bead_id}} --json`. beads-plan parses the change, enriches leaf tasks, and emits the apply-phase atoms as children of the formula step's bead. The returned root-epic ID is how the `apply` step finds its work.

4. **`beads-plan plan` is renamed to `beads-plan compile` and gains `--parent` and `--json` flags.** The name change signals that the command is a leaf tool — it compiles a change directory into apply-phase atoms, it does not orchestrate a workflow. `--parent <bead-id>` tells the compiler to create the root epic as a child of an existing bead rather than standalone. `--json` emits a machine-readable summary (`{root_id, leaf_ids, test_task_ids, tiers}`) so the formula step can graft the result into the molecule. No new subcommand is added; the old `plan` verb is removed in the same PR (no deprecation period — only formula steps and a few local scripts call it).

5. **Test-correction sub-molecule.** beads-plan detects test tasks (section titled "Tests", task containing the keyword `test`, or explicit inline `<!-- test -->` metadata) and compiles each one into a four-bead structure:
   - `test-task` (epic)
   - `execute` — do the work described by the task
   - `run-tests` — run the resulting tests
   - `correct` — created as an empty stub; the formula's retry pattern spawns a new `correct-N / run-tests-N` pair each time tests fail
   The retry loop itself — `run-tests fails → spawn correct → correct closes → re-run-tests` — lives in the formula as a sub-pattern, not in beads-plan Go code. The max-retry cap is a formula variable (`test_retry_cap`, default TBD per question 4). Exceeding the cap creates a `bd human` bead labeled `stuck-test`.

6. **Two new capability specs** describing (a) the formula contract — steps, gates, variables, phase — and (b) the test-task compile contract — what beads-plan emits when it detects a test task, and the JSON it returns via `--json`.

7. **Docs.** README update explaining how to start a new change using the formula (`bd mol pour meow-openspec --var change_dir=...`), a short runbook for operators describing each gate and what to inspect before resolving it, and a note deprecating any mental model where beads-plan is the orchestrator.

## Capabilities

### New Capabilities
- `meow-execution-formula`: The reusable MEOW formula that models the full OpenSpec lifecycle as a gated bead molecule, including phase-transition human gates and the test-retry sub-pattern.
- `meow-test-correction-loop`: The contract between beads-plan (which emits test-task sub-epics at compile time) and the formula (which drives the retry loop at runtime), including what status signals close the parent test-task epic.

### Modified Capabilities
- `beads-plan-cli`: the `plan` command is renamed to `compile` and gains two flags (`--parent`, `--json`). The core behavior — compile apply-phase atoms from a change directory — is unchanged.

## Impact

**Affected code:**
- `internal/cli/plan.go` → rename file to `compile.go`; rename the cobra `Use: "plan"` to `"compile"`; add `--parent` and `--json` flags; thread `--parent` through to the planner's root-epic creator; emit JSON summary when `--json` is set. Update `internal/cli/root.go` to register the renamed command.
- `internal/planner/planner.go` — `CreateRootEpic` gains an optional parent-ID argument; when set, the root is created with `--parent <id>` instead of standalone. Test tasks become a four-bead sub-epic instead of a flat leaf.
- `internal/planner/enrichment.go` — add `IsTest bool` on enriched tasks; detection layered as described.
- `internal/planner/bdclient.go` — no new methods required; `Create()` already supports `Parent`.
- New file: `.beads/formulas/meow-openspec.formula.toml`.
- New files: delta specs at `openspec/changes/meow-openspec-execution/specs/meow-execution-formula/spec.md` and `.../meow-test-correction-loop/spec.md`.

**Affected dependencies:**
- Requires `bd` with `formula`, `cook`, `mol pour`, `gate`, and `mol ready --gated` support (verified during research). A minimum-version check should live in the formula's `bd mol seed` prerequisites, not in beads-plan Go code.
- No new Go module dependencies expected.

**Affected operators:**
- New entry point: `bd mol pour meow-openspec --var change_dir=openspec/changes/<name>`. Operators who used `beads-plan plan …` directly must switch to `beads-plan compile …` (same behavior, new name).
- New concept: gate beads that must be resolved before the molecule advances. `bd human list` and `bd gate list` populate during an in-flight change; `bd mol current <id>` shows the current step.
- New artifact on disk: `.beads/formulas/meow-openspec.formula.toml` — checked into git so the template is shared across contributors.

**Not in scope for this change:**
- Implementation. This change proposes and designs the template; implementation lives in follow-up issues.
- Removing the compile-apply-phase leaf tool. It remains — just renamed from `plan` to `compile`.
- Multi-change orchestration (running several OpenSpec changes in one molecule). Separate, larger problem.
- CI integration, shared dashboards, run telemetry.
