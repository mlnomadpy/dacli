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
- [x] A dry run with an explicit active/proposed task window succeeds when the corresponding real ship could accept that task.
- [x] The preview names exactly the task refs the real post-accept wave would integrate without mutating task status.
- [x] A task that cannot be accepted remains a preview error with the same reason the real accept step would report.
- [x] A regression test proves the old preview rejects the fixture and the fixed preview emits accept then integrate for that explicit ref.
- [x] `--dry-run` performs no workspace, git, or GitHub mutation.
## Log
- 2026-08-27T13:07:47Z claimed by a-fixer-11h4hg
- 2026-08-27T13:25:22Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/828 (event 01M11NMWVSMBA2BT1BG9E35PYM)
- 2026-08-27T13:26:47Z accepted by a-root
- 2026-08-27T13:26:47Z verified by `GOCACHE=/tmp/dacli-go-cache-447 go test ./...` (exit 0) in branch dacli/447-fix-ship-dry-run-rejecting-an-active-explicit-task-window-that-real-ship-accepts at b3d5a4ba — proves that tree builds, not that the work is in trunk
- 2026-08-27T13:26:47Z deliverable: dacli/447-fix-ship-dry-run-rejecting-an-active-explicit-task-window-that-real-ship-accepts is merged into main
- 2026-08-27T13:26:47Z completed by a-root
## Verification Evidence
{"command":"GOCACHE=/tmp/dacli-go-cache-447 go test ./...","exit_code":0,"duration_ms":59530,"artifact_hash":"sha256:16586cacb1eb86ba9119ec807bf6e488d57fe90b850db1d9b1f44b1c8bdbc81a","verifier":"a-root","branch":"dacli/447-fix-ship-dry-run-rejecting-an-active-explicit-task-window-that-real-ship-accepts","commit_sha":"b3d5a4ba0c782d5e43af2ef0bdecd60f146eee9e"}
{"command":"GOCACHE=/tmp/dacli-go-cache-447 go test ./...","exit_code":0,"duration_ms":1602,"artifact_hash":"sha256:07db5a7f494d86162ce19dd327d134bb8ac313296b8ab5376f50a81125baa384","verifier":"a-root","branch":"dacli/447-fix-ship-dry-run-rejecting-an-active-explicit-task-window-that-real-ship-accepts","commit_sha":"b3d5a4ba0c782d5e43af2ef0bdecd60f146eee9e"}
