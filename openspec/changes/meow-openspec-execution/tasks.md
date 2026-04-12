## 1. Scaffolding and preconditions

- [ ] 1.1 Verify installed `bd` supports `formula`, `cook`, `mol pour`, `gate`, and `mol ready --gated`; record the minimum version in the formula prerequisites
- [ ] 1.2 Add a `.beads/formulas/` directory if missing and ensure it is tracked by git
- [ ] 1.3 Create an empty `openspec/changes/example/` smoke-test fixture with a minimal `proposal.md`, `design.md`, `tasks.md`, and one `specs/<cap>/spec.md` for use by formula tests

## 2. Rename `plan` to `compile`

- [ ] 2.1 Rename `internal/cli/plan.go` to `internal/cli/compile.go`; update the cobra `Use: "plan"` to `"compile"` and rewrite help text to describe a leaf-tool compiler
- [ ] 2.2 Update `internal/cli/root.go` to register the renamed command; remove any back-compat alias
- [ ] 2.3 Search the repo for references to `beads-plan plan` (README, skills, prime output, tests) and replace with `beads-plan compile`
- [ ] 2.4 Update `cmd/beads-plan/` if it hardcodes the old name anywhere
- [ ] 2.5 Run `make build` and confirm the binary exposes `compile` and no longer exposes `plan`

## 3. `--parent` flag on compile

- [ ] 3.1 Add `ParentID string` to the CreateRootEpic options in `internal/planner/planner.go`
- [ ] 3.2 Thread `ParentID` through `BdCLI.Create` in `internal/planner/bdclient.go` — pass `--parent <id>` to `bd create` when set
- [ ] 3.3 Add `--parent` flag to the cobra `compile` command and wire it into the planner options
- [ ] 3.4 Update the `BeadClient` mock in tests to record the Parent option for assertions

## 4. `--json` flag on compile

- [ ] 4.1 Define a `CompileSummary` struct in `internal/planner` with fields `RootID`, `LeafIDs`, `TestTaskIDs`, `Tiers map[string]string`
- [ ] 4.2 Populate `CompileSummary` as the planner creates beads; collect leaf IDs (excluding `run-tests-N` and `correct-N`), test-task epic IDs, and tier assignments
- [ ] 4.3 Add `--json` flag to the cobra `compile` command; on success, marshal `CompileSummary` to stdout as a single line; on failure, emit no JSON and let diagnostics go to stderr
- [ ] 4.4 Ensure JSON output is suppressed when `--json` is not passed, preserving today's human-readable output

## 5. Test-task detection

- [ ] 5.1 Add `IsTest bool` to the enriched task struct in `internal/planner/enrichment.go`
- [ ] 5.2 Implement the explicit-marker rule: match `<!-- test -->` anywhere after the checkbox, whitespace-tolerant
- [ ] 5.3 Implement the section-based rule: case-insensitive match of `/\btest(s|ing)?\b/` against the section title
- [ ] 5.4 Implement the keyword-fallback rule with the non-test section suppressor (`refactor|document|docs|rename`)
- [ ] 5.5 Apply detection in priority order during enrichment; record which rule fired in task metadata for debugging

## 6. Test-task sub-epic compile branch

- [ ] 6.1 In `internal/planner/planner.go`, branch on `IsTest` when creating a leaf task: instead of a single leaf bead, emit a four-bead sub-epic (`test-task` epic + `execute` + `run-tests-1` + `correct-1`)
- [ ] 6.2 Apply labels `meow:test`, `meow:test-run`, `meow:test-correct` to the four beads per the spec
- [ ] 6.3 Set `{"iteration": 1}` metadata on `run-tests-1` and `correct-1`
- [ ] 6.4 Preserve the single-task collapse rule around test tasks — the `test-task` epic skips the sub-epic wrapper when its section contains exactly one task
- [ ] 6.5 Ensure the `execute` bead carries the enriched description, acceptance, design context, spec-id, and tier that would have gone on a non-test leaf
- [ ] 6.6 Record the test-task epic ID in `CompileSummary.TestTaskIDs` and the `execute` bead ID in `CompileSummary.LeafIDs`

## 7. Formula file authoring

- [ ] 7.1 Author `.beads/formulas/meow-openspec.formula.toml` with `phase = "liquid"` and nine steps `explore → proposal → specs → design → tasks → plan → apply → verify → archive`
- [ ] 7.2 Declare variables `change_dir` (required), `change`, `skip_gates`, `add_gates`, `test_retry_cap` (default `1`) with the validation rules from the spec
- [ ] 7.3 Attach mandatory `gate: { type = "human" }` to `proposal`, `design`, `verify`, and `archive` steps
- [ ] 7.4 Attach default-on optional gates to `specs` and `tasks`, conditional on `skip_gates` not listing them
- [ ] 7.5 Attach conditional gate to `explore` when `add_gates` contains `explore`
- [ ] 7.6 Populate gate metadata with `{change, phase, artifact}` for every gate step
- [ ] 7.7 Wire step 6 (`plan`) with `exec = "beads-plan compile {{change_dir}} --parent {{self.bead_id}} --json"` and `capture = "apply_compile"`
- [ ] 7.8 Wire step 7 (`apply`) to depend on `{{steps.plan.capture.root_id}}` as a parent or depends-on edge
- [ ] 7.9 Implement the test-retry sub-pattern (watch `meow:test-run` close metadata, spawn `correct-(N+1)` on fail, then `run-tests-(N+1)`, respect `test_retry_cap`, escalate to `bd human` with label `meow:stuck-test`)
- [ ] 7.10 Reject mandatory-gate suppression and unknown variables at formula validation time

## 8. Formula smoke tests

- [ ] 8.1 Add a `make test-formula` target (or extend `make test`) that runs `bd mol seed meow-openspec --var change_dir=openspec/changes/example` and asserts no errors <!-- test -->
- [ ] 8.2 Add a test that pouring with `--var skip_gates=proposal` fails with a mandatory-gate validation error <!-- test -->
- [ ] 8.3 Add a test that pouring without `change_dir` fails with a required-variable error <!-- test -->
- [ ] 8.4 Add a test that a successful pour of the example fixture produces nine ordered steps and six open gates (four mandatory + two optional defaults) <!-- test -->

## 9. Compile tests

- [ ] 9.1 Add unit tests in `internal/planner/enrichment_test.go` for the three-layer test detection (explicit, section-based, keyword, non-test-section suppressor) <!-- test -->
- [ ] 9.2 Add a compile test that asserts a test task produces a four-bead sub-epic with the correct labels and edges (using a mock `BeadClient`) <!-- test -->
- [ ] 9.3 Add a compile test that asserts `--json` output matches the `CompileSummary` schema and that `execute` beads appear in `leaf_ids` while `run-tests-N` / `correct-N` do not <!-- test -->
- [ ] 9.4 Add a compile test that asserts `--parent` threads through to `bd create --parent <id>` on the root epic <!-- test -->
- [ ] 9.5 Add a compile test that asserts non-zero exit emits no JSON on stdout <!-- test -->

## 10. End-to-end dry run

- [ ] 10.1 Pour the formula against the example fixture, walk through each gate manually with `bd gate resolve`, and confirm `bd mol current` advances through every step to archive
- [ ] 10.2 Pour the formula against this very change (`meow-openspec-execution`) and verify the test tasks in sections 8 and 9 compile into test-task sub-epics <!-- test -->
- [ ] 10.3 Force a failing `run-tests-1` on a test-task from (10.2), confirm the retry loop spawns `correct-2` and `run-tests-2`, then force that to pass and confirm the parent `test-task` epic closes <!-- test -->
- [ ] 10.4 Force two consecutive failures with default `test_retry_cap=1`, confirm a `meow:stuck-test` human bead is created and `bd human list` surfaces it with the correct metadata <!-- test -->

## 11. Documentation and runbook

- [ ] 11.1 Update `README.md` to describe `bd mol pour meow-openspec --var change_dir=…` as the primary entry point and `beads-plan compile` as the leaf tool
- [ ] 11.2 Add a short runbook section under `README.md` (or a new `docs/meow-openspec-runbook.md`) listing each gate, what artifact to inspect, and what `bd` commands to run to resolve
- [ ] 11.3 Update `internal/cli/prime.go` output to mention the formula and the renamed `compile` command
- [ ] 11.4 Add a brief migration note: "`beads-plan plan` has been renamed to `beads-plan compile`"

## 12. Release hygiene

- [ ] 12.1 Run `make lint` and `make test` clean
- [ ] 12.2 Run `bd preflight` and resolve any lint/stale/orphan issues on this change's beads
- [ ] 12.3 Update `CHANGELOG` or equivalent if one exists
- [ ] 12.4 Walk this change through its own formula end-to-end (archive it via `bd gate resolve <archive-gate-id>` and `openspec archive meow-openspec-execution`) as the acceptance ritual
