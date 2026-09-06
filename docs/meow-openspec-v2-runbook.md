# meow-openspec-v2 runbook (draft)

Operator's guide for the v2 lifecycle formula. Differs from v1 in step order
(proposal+design together), adds the independent review, phase-bead compile,
self-containment lint, and the subagent dispatch protocol. The formula file:
`.beads/formulas/meow-openspec-v2.formula.toml`.

## Flow at a glance

```
idea ──▶ explore ──▶ draft-pd ──▶ [GATE human: pd] ──▶ independent-review ─┐
                                    ▲ (reopen on FAIL)                    │
 ◀────────────────────────────────────┘                                    ▼
                                  draft-st ──▶ [GATE human: st] ──▶ compile (phase beads)
                                              ──▶ self-containment-lint ──▶ [GATE human: molecule]
                                              ──▶ dispatch (subagents, feature branch)
                                              ──▶ verify (openspec --strict) ──▶ [GATE human: verify]
                                              ──▶ archive ──▶ [GATE human: final]
```

## Pouring

```sh
bd mol pour meow-openspec-v2 \
  --var change_dir=openspec/changes/my-change \
  --var change=my-change \
  --var branch=change/my-change
```

## The two "automagic" boundaries

1. **After the molecule gate** — no more human attention needed until a gate
   fires or an operator-input bead appears. The orchestrator supervises
   subagents; subagents surface needs via `bd human ask`.
2. **Verify gate** — human re-enters with evidence in hand, merges, and the
   archive runs on acknowledgement.

## Dispatch protocol (step 10)

Orchestrator (the pi session that poured the molecule):

1. `git checkout -b {{branch}}` — all artifacts and code live here.
2. Spawn executor subagent(s), **fresh context**, prompt exactly:
   > You are executing bead `<epic-id>` in `<repo>` on branch `{{branch}}`.
   > Workflow: `bd show <epic-id>` → `bd ready` → claim a ready child →
   > read its description fully → execute → run stated quality gates →
   > `bd close <id> --comment "<what you did, evidence>"`. Repeat until no
   > beads are ready. If a bead needs operator input, run
   > `bd human ask <id> -t "<question>"` and continue with other ready
   > work. If a bead's stated facts drift from reality, stop that lane and
   > report. Never commit to main; never force-push.
3. One agent walking the dependency chain by default; parallel lanes only
   when the molecule marks them independent.
4. Poll `bd ready`/notifications rather than sleep-loops; close the dispatch
   bead when all children are closed.

## Self-containment checklist (step 8)

A bead is self-contained iff a fresh agent with repo access only (no session
history) can execute it. Audit each bead for:

1. Authoritative artifact paths (openspec dir, specs) named.
2. Copy-pasteable commands, including run-from directory and full flags.
3. Host facts inline: which machine, ssh alias, IPs, ports, service names,
   file paths. ("mini = this Mac", "ssh optiplex-5090" style.)
4. No orphan references — nothing the earlier beads did not create.
5. Concrete expected values (counts, IDs, ports) so drift is detectable.
6. Operator-manual steps explicitly marked + surfaced via `bd human ask`.
7. Rollback/fallback stated wherever state moves or services restart.

Patch gaps with `bd update <id> -d "<fixed description>"`; never leave
"context TBD" in a bead. Evidence from the 2026-09-06 audit: 4 of 6 beads
had gaps against this list (orphan sops template, unstated host identity,
unexplained token provenance, vague NAS/archive references).

## Why phase beads, not leaf molecules

v1's compile produced 4-deep nested micro-beads (`…3.1.1.1`). Operators
could not review the molecule, and micro-beads lacked per-task context.
v2 compiles **one bead per tasks.md section** with the section's tasks inline
and a strict dependency chain — the structure that worked for
infra-1tc (monitoring → 5090). Use `--granularity phase`. Leaf granularity
remains available for genuinely parallel sections, but every leaf must
still pass the self-containment checklist.

## Known tool deltas needed

- `beads-plan compile --granularity phase` — phase-bead mode (currently
  compiles leaf molecules). Interim: hand-roll phase beads as in infra-1tc
  and note it in the compile bead.
- `bd reopen` used by the independent reviewer to loop drafting — verify
  against installed bd version (1.2.2).