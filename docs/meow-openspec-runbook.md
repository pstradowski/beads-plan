# meow-openspec formula runbook

A reference for operators walking an OpenSpec change through the `meow-openspec` beads formula — what each step is for, what each gate is blocking on, and the commands to unblock each one.

This runbook is the operator's counterpart to the formula file at `.beads/formulas/meow-openspec.formula.toml` and the design decisions in `openspec/changes/meow-openspec-execution/design.md`.

## Prerequisites

| Tool | Minimum | Check |
| --- | --- | --- |
| [bd](https://github.com/steveyegge/beads) | `1.0.0` | `bd version` |
| [openspec](https://github.com/steveyegge/openspec) | any recent | `openspec --version` |
| `beads-plan` | this repo | `beads-plan --help` |

Formula smoke test: `make test-formula` — asserts seed, required-variable enforcement, and the 14-step / 6-gate shape of the cooked formula. Run this before pouring for real.

### One-time setup: register the `gate` custom type

bd 1.0.0's formula pour machinery creates companion beads of type `gate` whenever a step has `gate = { type = "human" }`, but `gate` is not in bd's default issue-type whitelist (`bug|feature|task|epic|chore|decision`). Pour fails with `invalid issue type: gate` until you register it as a custom type:

```sh
bd config set types.custom '["gate"]'
```

This is persisted in the bd config database and only needs to be run once per machine. `make test-formula` checks for it and auto-applies if missing. Upstream bug tracked in `beads-plan-c45`.

## Pouring the molecule

For a new OpenSpec change called `my-change`:

```sh
# Create the OpenSpec change directory with openspec first, then:
bd mol pour meow-openspec \
    --var change_dir=openspec/changes/my-change \
    --var change=my-change
```

Record the returned molecule ID. You can find it later with `bd list --type=molecule`.

Both variables are required in practice. `change_dir` is schema-required; `change` has a placeholder default (`<change>`) that will make titles look wrong if you omit it.

## The 14-step lifecycle

The poured molecule contains fourteen ordered steps. Agents pick up ready beads via `bd ready`; the workflow pauses at six human review gates along the way.

```
  1.  explore            (no gate — skip by claiming and closing immediately if you already have context)
  2.  proposal           author proposal.md
  3.  proposal-review    HUMAN GATE — blocks specs
  4.  specs              author specs/<capability>/spec.md
  5.  specs-review       human gate — blocks design
  6.  design             author design.md
  7.  design-review      HUMAN GATE — blocks tasks
  8.  tasks              author tasks.md
  9.  tasks-review       human gate — blocks plan
  10. plan               run `beads-plan compile` with --parent and --json
  11. verify             run `openspec verify` and address issues
  12. verify-review      HUMAN GATE — blocks archive
  13. archive            run `openspec archive`
  14. archive-review     HUMAN GATE — final checkpoint
```

**HUMAN GATE** in caps = one of the four mandatory gates the user explicitly asked for (proposal, design, verify, archive). Lowercase "human gate" = the two additional v1 guardrails (specs, tasks) that v2 will make suppressible. In v1 all six behave identically — you have to close each one to proceed.

## Working through the workflow

### Finding the next thing to do

```sh
bd mol current <mol-id>          # show all steps with readiness
bd ready                         # list every unblocked bead in the repo
bd ready --parent <step-id>      # list children of a specific step
```

### Claiming work

```sh
bd update <step-id> --claim
bd update <step-id> --status=in_progress
```

### Reading a step's instructions

Every step in the `meow-openspec` formula carries its instructions in the description. Read them before touching anything:

```sh
bd show <step-id>
```

## Gate resolution: step-by-step

When you arrive at a review gate, these are the artifacts to read and the commands to run.

### 3. `proposal-review` (MANDATORY)

- **Read:** `openspec/changes/<change>/proposal.md`
- **Check for:** a clear "Why" (motivation for the change), a "What Changes" list that matches the intent, capabilities listed by kebab-case name, and a concrete Impact section. Reject if the proposal is vague or if the capabilities don't match what the spec files will actually describe.
- **Accept:** `bd gate resolve <gate-id>` — or `bd human respond <gate-id>` with a short ack comment.
- **Reject:** comment on the `proposal` step (not the gate) and reopen it with `bd update <proposal-step-id> --status=open`. The author re-drafts, then comes back to this gate.

### 5. `specs-review`

- **Read:** every `openspec/changes/<change>/specs/<capability>/spec.md` file.
- **Check for:** each requirement uses SHALL/MUST language, every requirement has at least one `#### Scenario:` with WHEN/THEN steps. No three-hashtag scenarios (OpenSpec requires four hashtags exactly — a common silent failure).
- **Accept:** `bd gate resolve <gate-id>`.

### 7. `design-review` (MANDATORY)

- **Read:** `openspec/changes/<change>/design.md`
- **Check for:** Context, Goals/Non-Goals, Decisions, Risks/Trade-offs. The decisions should reference the proposal and specs — if the design diverges from the proposal, either the proposal or the design needs to be revised. The risks section should be honest about what could go wrong.
- **Accept:** `bd gate resolve <gate-id>`.
- **Reject:** reopen `design` step, have the author revise.

### 9. `tasks-review`

- **Read:** `openspec/changes/<change>/tasks.md`
- **Check for:** tasks use `- [ ] X.Y description` checkbox format under `## N. Group` headers, tasks are small enough for one session, tasks are ordered by dependency, and every genuinely-test task carries a trailing `<!-- test -->` annotation so `beads-plan compile` wraps it in a four-bead sub-epic.
- **Accept:** `bd gate resolve <gate-id>`.

### 10. `plan` (not a gate — a work step)

This step has no gate but it's the heart of the workflow. Claim it, then run the command in its description:

```sh
beads-plan compile openspec/changes/<change> --parent <plan-step-id> --json
```

The command emits a single-line JSON summary on stdout. Record the `root_id` on the plan step so downstream steps can find it:

```sh
bd update <plan-step-id> --metadata '{"apply_compile_root": "<root_id-from-json>"}'
```

The plan step is declared with `waits_for = all-children`, so it stays `in_progress` while you work through the compiled apply-phase leaves via `bd ready`. When every compiled child closes, the plan step becomes eligible to close and unblocks `verify`.

### 12. `verify-review` (MANDATORY)

- **Read:** the report produced by step 11 (`openspec verify <change>`), plus a spot-check of a handful of the compiled apply-phase beads (`bd show <some-leaf-id>`).
- **Check for:** no CRITICAL issues in the verify report, every task in tasks.md is checked off, every test-task epic closed with a passing `run-tests-N` bead, no orphaned follow-up issues that should have been part of this change.
- **Accept:** `bd gate resolve <gate-id>`.

### 14. `archive-review` (MANDATORY — final checkpoint)

- **Read:** confirm `openspec/changes/<change>/` moved to `openspec/changes/archive/YYYY-MM-DD-<change>/`. Confirm delta specs were synced into `openspec/specs/<capability>/spec.md` (or that the sync was explicitly skipped for a good reason).
- **Check for:** follow-up issues filed for anything you noticed but did not fix, the change log updated, tests and linters clean.
- **Accept:** `bd gate resolve <gate-id>`. The molecule closes automatically once this final step closes.

## Running down the retry loop for a failing test task

In v1, the test-correction loop is **manual**. When you see `run-tests-N` close with `outcome=fail`:

1. Look at the parent `test-task` epic and its sibling beads (`bd show <test-task-epic-id>`).
2. Claim `correct-1` (or the next correct bead in the sequence), fix the underlying problem, and close it.
3. Manually create a new `run-tests-2` bead under the same test-task epic:
   ```sh
   bd create \
       --title "<task-num> run-tests-2" \
       --type task \
       --parent <test-task-epic-id> \
       -l meow:test-run \
       --metadata '{"iteration":"2"}'
   bd dep add <new-run-tests-2-id> <correct-1-id>
   ```
4. Work through `run-tests-2`. If it passes, close the parent `test-task` epic. If it fails, repeat from step 2 with `correct-2` / `run-tests-3`, etc.

v2 will automate this loop (tracked in `beads-plan-c45`). Until then, the four-bead compile-time structure is there but an agent has to drive the retry cycle themselves.

## Recovering a stuck molecule

| Symptom | Diagnosis | Fix |
| --- | --- | --- |
| `bd ready` returns no molecule steps but the molecule isn't closed | Likely a gate is open that hasn't been closed | `bd gate list` to see open gates; resolve the right one |
| A review gate closes but the next step stays blocked | `bd mol ready --gated` hasn't re-evaluated | Run `bd mol ready --gated` to force re-evaluation |
| A step is ready but no one picks it up | Assignee missing, or tier mismatch | `bd update <step-id> --claim` manually |
| `plan` step can't find its compiled children | The operator forgot to pass `--parent <plan-bead-id>` | Delete the orphaned compile output, re-run with `--parent` set correctly |

## Related commands

```sh
bd formula show meow-openspec        # formula metadata
bd mol show <mol-id>                 # molecule structure
bd mol current <mol-id>              # current position in workflow
bd mol progress <mol-id>             # progress summary
bd gate list                         # open gates
bd human list                        # human-labeled open beads (review gates show up here)
bd preflight                         # pre-archive sanity check
```
