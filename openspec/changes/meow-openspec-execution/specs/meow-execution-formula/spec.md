## ADDED Requirements

### Requirement: Formula file location and phase

The system SHALL ship a beads formula at `.beads/formulas/meow-openspec.formula.toml`, checked into git, declaring `phase = "liquid"`.

#### Scenario: Formula is discoverable by bd
- **WHEN** an operator runs `bd formula list` from the repository root
- **THEN** `meow-openspec` appears in the output

#### Scenario: Formula compiles cleanly via seed
- **WHEN** an operator runs `bd mol seed meow-openspec --var change_dir=openspec/changes/example`
- **THEN** `bd` reports the formula is accessible with no schema validation errors

#### Scenario: Formula declares liquid phase
- **WHEN** an operator inspects the formula file or runs `bd formula show meow-openspec`
- **THEN** the `phase` field is `liquid`, signalling that it should be instantiated via `bd mol pour` (persistent) rather than `bd mol wisp` (ephemeral), because OpenSpec changes have long-term audit value

### Requirement: Lifecycle step structure

The formula SHALL define fourteen ordered steps that together form the full OpenSpec lifecycle. The lifecycle comprises seven work phases — `explore`, `proposal`, `specs`, `design`, `tasks`, `plan`, `verify`, `archive` — and six human review gates interleaved after the artifact-producing phases. Each step SHALL produce a bead in the molecule when the formula is poured.

#### Scenario: Poured molecule lists every lifecycle step in order
- **WHEN** an operator runs `bd mol pour meow-openspec --var change_dir=openspec/changes/example --var change=example` and then `bd mol current <mol-id>`
- **THEN** the output lists the fourteen steps in the order `explore → proposal → proposal-review → specs → specs-review → design → design-review → tasks → tasks-review → plan → verify → verify-review → archive → archive-review`

#### Scenario: Steps are linearly ordered by dependencies
- **WHEN** the molecule is inspected via `bd mol show <mol-id>`
- **THEN** each step depends on the previous step via `depends_on`, so that no step can start until its predecessor closes

#### Scenario: Plan step uses waits_for=all-children
- **WHEN** the formula is cooked to JSON via `bd cook meow-openspec --mode runtime --var change_dir=… --var change=…`
- **THEN** the `plan` step's `waits_for` field is set to `all-children`, meaning the step stays in_progress while an agent works through the compiled apply-phase atoms (the children grafted under it by `beads-plan compile --parent <plan-bead-id>`)

### Requirement: Mandatory human review gates

The formula SHALL attach a `gate = { type = "human" }` field to exactly six review steps: `proposal-review`, `specs-review`, `design-review`, `tasks-review`, `verify-review`, and `archive-review`. All six are mandatory in v1 — the formula cannot be instantiated in a way that skips any of them.

#### Scenario: Six review gates are created on pour
- **WHEN** an operator pours the formula and runs `bd gate list` filtered to the molecule
- **THEN** exactly six open human gates appear, one per review step

#### Scenario: Each mandatory gate blocks its own step
- **WHEN** a review step's upstream work step has closed but the review gate remains open
- **THEN** `bd mol current <mol-id>` shows the molecule paused at that review step, and the next work step (e.g. `specs` after `proposal-review`) stays blocked

#### Scenario: Resolving a gate unblocks downstream
- **WHEN** a reviewer runs `bd gate resolve <gate-id>` (or `bd human respond <id>`) on an open review gate
- **AND** then runs `bd mol ready --gated`
- **THEN** the molecule appears in the ready-gated list and the blocked downstream work step transitions to `ready`

#### Scenario: Mandatory gates cannot be suppressed via variables
- **WHEN** an operator attempts to pour with a speculative variable like `--var skip_gates=proposal`
- **THEN** the variable is accepted by the schema (since v1 does not implement suppression) but the `proposal-review` gate is created anyway; the formula documentation in the file header explicitly states that v1 treats all six gates as mandatory

### Requirement: Plan step carries compile instructions

Because the beads 1.0.0 formula schema has no `exec` field on steps, the `plan` step (step 10) SHALL carry instructions in its `description` telling whichever agent claims it to run `beads-plan compile {{change_dir}} --parent <this-bead-id> --json` and record the returned `root_id` in the bead's metadata via `bd update`.

#### Scenario: Plan step description names the compile command
- **WHEN** a reviewer inspects the `plan` step via `bd show <plan-bead-id>`
- **THEN** the description contains the literal command `beads-plan compile` and explains the `--parent` and `--json` flags

#### Scenario: Compiled root is grafted under the plan bead
- **WHEN** an agent runs `beads-plan compile <change-dir> --parent <plan-bead-id> --json`
- **THEN** the resulting root epic has `plan-bead-id` as its parent, so `bd show <plan-bead-id>` lists the compiled apply-phase epic among its children

#### Scenario: Plan bead closes when all compiled children close
- **WHEN** all compiled apply-phase leaf beads (and test-task epics) have closed
- **AND** the `plan` step bead has `waits_for = all-children`
- **THEN** the `plan` step is eligible for close, and closing it unblocks the `verify` step

### Requirement: Formula variable surface

The formula SHALL accept two variables and SHALL reject pour invocations that omit the required one:

| Variable     | Required | Default    | Description                                                                 |
| ------------ | -------- | ---------- | --------------------------------------------------------------------------- |
| `change_dir` | yes      | (none)     | Path to the OpenSpec change directory, e.g. `openspec/changes/my-change`    |
| `change`     | no       | `<change>` | Change identifier used in bead titles and descriptions                      |

Additional variables (`skip_gates`, `add_gates`, `test_retry_cap`) are **not** implemented in v1. They may be added in a follow-up once the beads formula schema supports conditional steps and step-level validation.

#### Scenario: Missing required variable fails
- **WHEN** an operator runs `bd mol pour meow-openspec` without `--var change_dir=…`
- **THEN** the command fails with a validation error naming `change_dir` as a required variable, and no molecule is created

#### Scenario: Default change identifier is a placeholder
- **WHEN** an operator pours with `--var change_dir=openspec/changes/example` and no explicit `--var change=…`
- **THEN** the cooked step titles contain the literal string `<change>` in place of a real identifier — operators are expected to pass `--var change=example` for every real pour

#### Scenario: Unknown variables are accepted without error
- **WHEN** an operator runs `bd mol pour meow-openspec --var change_dir=… --var skip_gates=specs`
- **THEN** the command succeeds but `skip_gates` has no effect — the `specs-review` gate is still created; the formula's file header documents this as a v1 limitation

### Requirement: Review context lives in step descriptions

Because the beads 1.0.0 formula schema has no structured metadata field on steps, every review-gate step SHALL carry in its `description` the information a human needs to resolve the gate:

- which artifact to read (e.g. `openspec/changes/<name>/proposal.md`),
- what to check for in that artifact,
- the `bd gate resolve` command to run on accept,
- what to do if the review finds problems.

#### Scenario: Review step description names the artifact
- **WHEN** a reviewer runs `bd show <proposal-review-bead-id>`
- **THEN** the description contains `{{change_dir}}/proposal.md` (substituted to the real path after pour) and explicit instructions for resolving the gate

#### Scenario: Review step description names the resolve command
- **WHEN** a reviewer reads the description of any review step
- **THEN** the description contains the literal command `bd gate resolve` so that non-expert reviewers know how to unblock the workflow
