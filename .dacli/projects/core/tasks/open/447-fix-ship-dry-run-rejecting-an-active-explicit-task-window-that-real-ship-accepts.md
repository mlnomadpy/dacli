---
id: t-01KZYREM23X0ADW8MDV26C1H9A
kind: task
created: 2026-08-13T23:48:51Z
created_by: a-root
owner: a-root
github:
  issue: 651
  repo: mlnomadpy/dacli
estimate: "{optimistic: 1, probable: 2, pessimistic: 3}"
---
# Fix ship dry-run rejecting an active explicit task window that real ship accepts
## Context
Adopted from GitHub issue #651.

## Symptom

For an active task proposed for acceptance, the real command can run:

```bash
dacli ship --project core --tasks <ref> --verify "<cmd>" --into main --pr
```

The real pipeline first runs `accept --all --force --defer-landing`, then resolves the explicit `--tasks` wave and integrates it. The matching preview instead resolves `explicitWave` immediately and refuses because the task is not yet `done`:

```text
--tasks: NNN-slug is active, not done — ship integrates done tasks branches
```

Thus `--dry-run` cannot preview the exact real command for the normal accept-then-integrate workflow.

## Suspected cause

`printPlan` calls `explicitWave` before the simulated accept step, while `cmdShip` calls `shipWave` after the real accept step. The preview validates pre-transition state against a post-transition invariant.

## Manual workaround

Preview `ship --project <slug> ... --dry-run` without `--tasks`, inspect the proposed wave separately, then run the real command with the explicit task window. This loses the strongest property of dry-run: exact scope parity.

## Acceptance criteria

- [ ] A dry run with an explicit active/proposed task window succeeds when the corresponding real ship could accept that task.
- [ ] The preview names exactly the task refs the real post-accept wave would integrate without mutating task status.
- [ ] A task that cannot be accepted remains a preview error with the same reason the real accept step would report.
- [ ] A regression test proves the old preview rejects the fixture and the fixed preview emits accept then integrate for that explicit ref.
- [ ] `--dry-run` performs no workspace, git, or GitHub mutation.

## Acceptance
- [ ] A dry run with an explicit active/proposed task window succeeds when the corresponding real ship could accept that task.
- [ ] The preview names exactly the task refs the real post-accept wave would integrate without mutating task status.
- [ ] A task that cannot be accepted remains a preview error with the same reason the real accept step would report.
- [ ] A regression test proves the old preview rejects the fixture and the fixed preview emits accept then integrate for that explicit ref.
- [ ] `--dry-run` performs no workspace, git, or GitHub mutation.
## Log
