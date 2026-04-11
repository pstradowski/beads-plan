## ADDED Requirements

### Requirement: Formula file location and phase

The system SHALL ship a beads formula at `.beads/formulas/meow-openspec.formula.toml`, checked into git, declaring `phase = "liquid"`.

#### Scenario: Formula is discoverable by bd
- **WHEN** an operator runs `bd formula list` from the repository root
- **THEN** `meow-openspec` appears in the output, with its source path resolving to `.beads/formulas/meow-openspec.formula.toml`

#### Scenario: Pouring as vapor produces a warning
- **WHEN** an operator runs `bd mol wisp meow-openspec --var change_dir=openspec/changes/example`
- **THEN** `bd` emits a warning that the formula's declared phase is `liquid` and the wisp invocation is discouraged, but the command still executes

#### Scenario: Formula compiles cleanly via seed
- **WHEN** an operator runs `bd mol seed meow-openspec --var change_dir=openspec/changes/example`
- **THEN** `bd` reports the formula is accessible and can be cooked, with no schema validation errors

### Requirement: Lifecycle step structure

The formula SHALL define exactly nine ordered lifecycle steps with the following identifiers and sequence: `explore`, `proposal`, `specs`, `design`, `tasks`, `plan`, `apply`, `verify`, `archive`. Each step SHALL produce a bead in the molecule when the formula is poured.

#### Scenario: Poured molecule contains all nine steps
- **WHEN** an operator runs `bd mol pour meow-openspec --var change_dir=openspec/changes/example` and then `bd mol current <mol-id>`
- **THEN** the output lists nine steps in the order `explore → proposal → specs → design → tasks → plan → apply → verify → archive`

#### Scenario: Steps are linearly ordered by dependencies
- **WHEN** the molecule is poured and inspected via `bd mol show <mol-id>`
- **THEN** each step depends on the previous step, so that `apply` cannot start until `plan` closes, `plan` cannot start until `tasks` closes, and so on back to `explore`

### Requirement: Mandatory human gates

The formula SHALL attach a `gate: { type: "human" }` field to the `proposal`, `design`, `verify`, and `archive` steps. These four gates SHALL NOT be suppressible via formula variables, and attempts to do so SHALL fail formula validation.

#### Scenario: Mandatory gates are created on pour
- **WHEN** an operator pours the formula and runs `bd gate list`
- **THEN** four open human gates appear, one each for phases `proposal`, `design`, `verify`, and `archive`

#### Scenario: Mandatory gate blocks its step
- **WHEN** the `proposal` step has been completed but its `proposal-accepted` gate remains open
- **THEN** the `specs` step stays in `blocked` status and `bd mol current` shows the molecule paused at the `proposal` gate

#### Scenario: Attempt to suppress mandatory gate fails
- **WHEN** an operator runs `bd mol pour meow-openspec --var change_dir=openspec/changes/example --var skip_gates=proposal`
- **THEN** the command fails with a formula validation error stating that `proposal` is a mandatory gate and cannot be suppressed, and no molecule is created

#### Scenario: Resolving a mandatory gate unblocks the next step
- **WHEN** an operator runs `bd gate resolve <proposal-gate-id>` (or `bd human respond <id>`)
- **AND** then runs `bd mol ready --gated`
- **THEN** the molecule appears in the ready-gated list and its `specs` step transitions from `blocked` to `ready`

### Requirement: Optional gates on specs and tasks

The formula SHALL attach `gate: { type: "human" }` to the `specs` and `tasks` steps by default. These gates SHALL be suppressible per pour via `--var skip_gates=<comma-separated>` where values are drawn from the set `{specs, tasks}`.

#### Scenario: Optional gates are on by default
- **WHEN** an operator pours the formula without passing `skip_gates`
- **THEN** `bd gate list` shows open human gates for phases `specs` and `tasks` in addition to the four mandatory gates

#### Scenario: Optional gate can be suppressed
- **WHEN** an operator runs `bd mol pour meow-openspec --var change_dir=… --var skip_gates=specs,tasks`
- **THEN** the poured molecule has no gates on the `specs` or `tasks` steps, and those steps proceed to the next step as soon as their own work closes

#### Scenario: Unknown gate name fails validation
- **WHEN** an operator runs `bd mol pour meow-openspec --var change_dir=… --var skip_gates=madeup`
- **THEN** the command fails with a formula validation error listing the recognised optional gate names, and no molecule is created

### Requirement: Optional explore gate

The formula SHALL leave the `explore` step un-gated by default, and SHALL accept a variable `add_gates` with value `explore` to attach a human gate to that step on demand.

#### Scenario: Explore is un-gated by default
- **WHEN** an operator pours the formula without passing `add_gates`
- **THEN** `bd gate list` shows no gate on the `explore` step

#### Scenario: Explore gate can be added on demand
- **WHEN** an operator runs `bd mol pour meow-openspec --var change_dir=… --var add_gates=explore`
- **THEN** `bd gate list` shows an open `explore-accepted` human gate blocking the `proposal` step

### Requirement: Gate bead metadata

Every gate bead created by the formula SHALL carry metadata in the shape `{"change": "<name>", "phase": "<phase-id>", "artifact": "<path-to-artifact>"}` where `<phase-id>` is one of `proposal`, `specs`, `design`, `tasks`, `verify`, `archive`, or `explore`.

#### Scenario: Gate bead is discoverable by phase
- **WHEN** an operator runs `bd human list --json` while the molecule is paused at the `design` gate
- **THEN** the returned gate bead has metadata field `phase` equal to `design` and `artifact` equal to `openspec/changes/<name>/design.md`

#### Scenario: Gate bead identifies its change
- **WHEN** the reviewer runs `bd show <gate-id> --json`
- **THEN** the bead's metadata `change` field equals the basename of the `change_dir` passed at pour time

### Requirement: Plan step invokes beads-plan compile

The `plan` step (step 6) SHALL have an `exec` field that invokes `beads-plan compile {{change_dir}} --parent {{self.bead_id}} --json` and SHALL capture the resulting JSON under the name `apply_compile` so that subsequent steps can reference `{{steps.plan.capture.root_id}}`.

#### Scenario: Plan step runs the compiler
- **WHEN** the molecule reaches step 6 and the `plan` step becomes ready
- **THEN** the exec command resolves to `beads-plan compile openspec/changes/<name> --parent <plan-bead-id> --json` and runs to successful exit

#### Scenario: Apply step can reference the compiled root
- **WHEN** the `plan` step closes with a captured JSON summary containing a non-empty `root_id`
- **AND** the `apply` step becomes ready
- **THEN** the apply step's bead is either a parent of, or has a depends-on edge to, the bead ID recorded in `apply_compile.root_id`

#### Scenario: Plan step failure does not advance the molecule
- **WHEN** `beads-plan compile` exits with a non-zero status during the `plan` step
- **THEN** the `plan` step stays in `in_progress` (or transitions to `failed` per beads' conventions), and the `apply` step stays blocked

### Requirement: Formula variable surface

The formula SHALL accept the following variables, and SHALL reject pour invocations that violate their constraints:

| Variable          | Required | Type    | Constraint                                                                  |
| ----------------- | -------- | ------- | --------------------------------------------------------------------------- |
| `change_dir`      | yes      | string  | Must be an existing directory containing at least `proposal.md`             |
| `change`          | no       | string  | Derived from `basename(change_dir)` if not set                              |
| `skip_gates`      | no       | string  | Comma-separated subset of `{specs, tasks}`                                  |
| `add_gates`       | no       | string  | Comma-separated subset of `{explore}`                                       |
| `test_retry_cap`  | no       | int     | Non-negative integer, default `1` (see meow-test-correction-loop spec)      |

#### Scenario: Missing required variable fails
- **WHEN** an operator runs `bd mol pour meow-openspec` with no `--var change_dir=…`
- **THEN** the command fails with a validation error naming `change_dir` as a required variable, and no molecule is created

#### Scenario: Non-existent change_dir fails
- **WHEN** an operator runs `bd mol pour meow-openspec --var change_dir=openspec/changes/does-not-exist`
- **THEN** the command fails with a validation error stating the directory does not exist or does not contain a `proposal.md`

#### Scenario: Unknown variable is rejected
- **WHEN** an operator runs `bd mol pour meow-openspec --var change_dir=… --var foo=bar`
- **THEN** the command fails with a validation error listing `foo` as an unknown variable

#### Scenario: Default-derived change variable
- **WHEN** an operator pours the formula with `change_dir=openspec/changes/example` and no explicit `change`
- **THEN** gate metadata and step descriptions in the poured molecule use `example` as the change identifier
