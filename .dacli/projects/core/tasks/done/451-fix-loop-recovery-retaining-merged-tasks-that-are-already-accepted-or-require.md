---
id: t-01KZYW7M979TQNHD2VTA1Q9WAT
kind: task
created: 2026-08-14T00:54:56Z
created_by: a-root
owner: a-root
priority: must
github:
  issue: 661
  repo: mlnomadpy/dacli
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
---
# Fix loop recovery retaining merged tasks that are already accepted or require verifier evidence
## Context
Adopted from GitHub issue #661.

## Symptom

After tasks 449 and 450 were merged, owner-verified with `dacli accept <ref> --verify "GOCACHE=/tmp/... go test ./..."`, and materialized as done with every acceptance box checked, each subsequent bounded loop still printed:

```
resuming: 2 task(s) awaiting merge confirmation
449: PR merged — closing the task record
449: PR merged but accept FAILED ... command acceptance criterion; pass --verify
450: PR merged — closing the task record
450: PR merged but accept FAILED ... command acceptance criterion; pass --verify
```

The done task records remained truthful, but the same stale recovery entries were retained across cycles 89 and 90.

## Proven cause

`internal/features/orchestration.reconcilePendingAccepts` restores `pending_accept` entries from the cycle journal and, for every merged PR, unconditionally runs `accept <seq> --force`. It does not first resolve the task and recognize an already-done/fully-accepted record, and the journal carries no verifier command/evidence for an open task whose criteria require `--verify`. On refusal it appends the entry back to `remaining`, making the warning permanent.

## Manual workaround

The root owner verified and accepted the tasks explicitly. The loop still retained the stale entries; there is no safe public command observed that removes only those reconciled journal entries.

## Design

Make reconciliation state-aware and idempotent. A merged journal entry whose task is already done and whose landing is confirmed should be cleared without re-acceptance. An open merged task that requires command evidence must surface a structured recovery action that preserves the entry until the owner supplies verifier evidence; it must not retry the same policy-refused command every cycle.

Implementation and claim boundary: `internal/features/orchestration` owns reconciliation, cycle-journal persistence, and the regression fixtures. Consume task status and acceptance evidence through existing shared APIs; file a separate narrow task if those APIs prove insufficient.

## Acceptance criteria

- [ ] A regression test persists a `pending_accept` entry for a merged PR whose task is already done with acceptance evidence, then proves one reconciliation clears the entry without invoking `accept`.
- [ ] The cleared entry is absent from the rewritten cycle journal and does not reappear on the next loop invocation.
- [ ] The already-done task is counted at most once in rollups and its acceptance/log evidence is not duplicated.
- [ ] A separate fixture covers an open merged task with a command acceptance criterion and no verifier evidence.
- [ ] That open fixture remains recoverable, emits one actionable message naming the required `--verify` remedy, and does not repeatedly invoke the same exit-3 command within later cycles.
- [ ] Journal recovery remains backward compatible with existing entries or a documented migration handles the format change.
- [ ] Mutation evidence proves the old unconditional `accept <seq> --force` behavior fails the regression.
- [ ] Focused orchestration tests and `go test ./...` pass.

## Acceptance
- [x] A regression test persists a `pending_accept` entry for a merged PR whose task is already done with acceptance evidence, then proves one reconciliation clears the entry without invoking `accept`.
- [x] The cleared entry is absent from the rewritten cycle journal and does not reappear on the next loop invocation.
- [x] The already-done task is counted at most once in rollups and its acceptance/log evidence is not duplicated.
- [x] A separate fixture covers an open merged task with a command acceptance criterion and no verifier evidence.
- [x] That open fixture remains recoverable, emits one actionable message naming the required `--verify` remedy, and does not repeatedly invoke the same exit-3 command within later cycles.
- [x] Journal recovery remains backward compatible with existing entries or a documented migration handles the format change.
- [x] Mutation evidence proves the old unconditional `accept <seq> --force` behavior fails the regression.
- [x] Focused orchestration tests and `go test ./...` pass.
## Log
- 2026-08-16T17:24:55Z claimed by a-maintainer-psevtg
- 2026-08-16T17:40:22Z accepted by a-root
- 2026-08-16T17:40:22Z verified by `GOCACHE=/tmp/dacli-451-final go test ./...` (exit 0) in branch main at dde0fd7 — proves that tree builds, not that the work is in trunk
- 2026-08-16T17:40:22Z deliverable: dacli/451-fix-loop-recovery-retaining-merged-tasks-that-are-already-accepted-or-require is merged into main
- 2026-08-16T17:40:22Z completed by a-root
- 2026-08-16T17:50:44Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/676 (event 01M05T1Y9AT69531QJ9DVD1A75)
## Verification Evidence
{"command":"GOCACHE=/tmp/dacli-451-accept go test ./...","exit_code":0,"duration_ms":73673,"artifact_hash":"sha256:5cc7fa92e0e17393517b1837b1c1aac16c52b58b41e88c424399ccb2bc475566","verifier":"a-root","branch":"main","commit_sha":"dde0fd7014834d5f4bd9c7941986ca710beba323"}
{"command":"GOCACHE=/tmp/dacli-451-final go test ./...","exit_code":0,"duration_ms":76950,"artifact_hash":"sha256:4e01a721cc5fc7aadf78a073e415b3a9a0bf68da7f568fa518e4c19bf099134e","verifier":"a-root","branch":"main","commit_sha":"dde0fd7014834d5f4bd9c7941986ca710beba323"}
