# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this project does

beads-plan converts OpenSpec task plans into executable beads workflows. It reads OpenSpec artifacts (proposal.md, design.md, specs/, tasks.md) from a change directory and produces a nested beads molecule with epics, enriched leaf tasks, dependency edges, complexity tiers, and parallelism metadata. It shells out to the `bd` CLI for all bead operations.

## Build commands

```sh
make build      # Build to ./build/beads-plan
make test       # Run all tests with -v
make lint       # Run golangci-lint
make install    # Install to GOPATH/bin
```

Run a single test:
```sh
go test ./internal/planner/ -run TestComplexity -v
```

Pre-commit hook runs `go vet` and `gofmt` check automatically.

## Architecture

The codebase follows a pipeline: **parse → enrich → plan → create beads**.

- **`cmd/beads-plan/`** — Entry point, delegates to `internal/cli`.
- **`internal/cli/`** — Cobra command definitions (`plan`, `view`, `prime`). Orchestrates the pipeline for each command.
- **`internal/parser/`** — Parses OpenSpec markdown. `markdown.go` handles tasks.md (sections + checkbox items). `artifacts.go` reads proposal/design/specs for context.
- **`internal/planner/`** — Core logic:
  - `planner.go` — Orchestrates bead creation: root epic → sub-epics (with single-task collapse) → leaf tasks → dependency edges.
  - `bdclient.go` — `BeadClient` interface + `BdCLI` implementation that shells out to `bd` binary. All bead mutations go through this interface.
  - `complexity.go` — Keyword-based complexity assessment (low/medium/high).
  - `tier.go` — Maps complexity to capability tiers (fast/standard/advanced).
  - `enrichment.go` — Enriches tasks with proposal context, design decisions, acceptance criteria, and output schema.
  - `parallelism.go` — Analyzes which tasks/sections can run concurrently.
- **`internal/config/`** — TOML config loading for provider profiles (`.beads-plan.toml`). Maps abstract tiers to concrete model names.
- **`internal/viewer/`** — Reverse direction: reads a beads epic hierarchy via `bd show --json` and renders it back to tasks.md format.

## Key patterns

- **BeadClient interface** (`internal/planner/bdclient.go`): All `bd` CLI interactions are behind `BeadClient`. Tests use mock implementations — never shell out to `bd` in tests.
- **Single-task collapse**: Sections with exactly one task skip the sub-epic wrapper and create the task directly under the root epic.
- **Metadata as JSON**: The `--metadata` flag to `bd create` takes a single JSON string, not key=value pairs.

## Git workflow

Git Flow: `main` (releases), `develop` (integration), `feature/*` (new work from develop).


<!-- BEGIN BEADS INTEGRATION v:1 profile:minimal hash:ca08a54f -->
## Beads Issue Tracker

This project uses **bd (beads)** for issue tracking. Run `bd prime` to see full workflow context and commands.

### Quick Reference

```bash
bd ready              # Find available work
bd show <id>          # View issue details
bd update <id> --claim  # Claim work
bd close <id>         # Complete work
```

### Rules

- Use `bd` for ALL task tracking — do NOT use TodoWrite, TaskCreate, or markdown TODO lists
- Run `bd prime` for detailed command reference and session close protocol
- Use `bd remember` for persistent knowledge — do NOT use MEMORY.md files

## Session Completion

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   bd dolt push
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds
<!-- END BEADS INTEGRATION -->
