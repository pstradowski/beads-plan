## Context

Smoke-test fixture. Not a real design.

## Goals / Non-Goals

**Goals**
- Provide a minimal valid OpenSpec change that the `meow-openspec` formula and `beads-plan compile` can exercise without polluting the real project's bead molecule.

**Non-Goals**
- Anything else.

## Decisions

- Fixture lives at `openspec/changes/example/` and is excluded from `openspec archive` runs.

## Risks / Trade-offs

- None. Fixture is throwaway.
