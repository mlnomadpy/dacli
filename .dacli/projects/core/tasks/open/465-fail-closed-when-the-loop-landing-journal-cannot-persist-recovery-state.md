---
id: t-01M0AEG5F23TRH6BAR9HT38ZP1
kind: task
created: 2026-08-18T12:45:49Z
created_by: a-root
owner: a-root
priority: must
github:
  issue: 687
  repo: mlnomadpy/dacli
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
---
# Fail closed when the loop landing journal cannot persist recovery state
## Context
Adopted from GitHub issue #687.

## Problem

The loop's landing journal is load-bearing but every persistence failure is discarded. `writeCycleJournal` returns no error and ignores failures from `MkdirAll`, removal of an empty journal, and `writeStateFile`. Its caller therefore continues after failing to persist `pending_accept`, `pending_land`, the token ceiling, or the resolved landing policy.

This contradicts the journal's own contract: losing it can rebuild already-in-flight work, forget acceptance recovery, release a held record push, or silently restore a run without its token ceiling. The governor snapshot is described as convenience state, but the landing journal is explicitly a recovery ledger.

## Design

Separate advisory snapshots from correctness-critical recovery records in the API. Make journal persistence return an error and make every transition that depends on the new journal state stop before reporting success. Persist atomically, validate the post-write record when appropriate, and handle empty-ledger removal as a checked state transition. Preserve idempotent restart behavior.



## Evidence

- `internal/features/orchestration/journal.go:70-102` discards every persistence error.
- `internal/features/orchestration/orchestration.go:545` calls it without a result.
- The journal header documents the exact safety properties lost on restart.

## Acceptance
- [ ] `writeCycleJournal` returns an error for directory creation, atomic write/rename, and empty-ledger removal failures.
- [ ] Every production caller propagates the error and the loop cannot report a successful checkpoint/cycle after the recovery ledger failed to persist.
- [ ] Fault-injection tests cover failed create, write/rename, and removal while pending accepts, pending lands, token ceiling, and landing policy are non-empty.
- [ ] A restart regression proves a failed checkpoint cannot silently rebuild an in-flight task or run uncapped.
- [ ] Advisory `loop status` snapshot failures remain explicitly best-effort and cannot be confused with journal durability.
- [ ] Mutation evidence shows the regression goes green if the returned error is discarded.
- [ ] Focused orchestration tests, race tests, and `go test ./...` pass.
## Log
- 2026-08-18T14:38:45Z claimed by a-maintainer-dgyp5f
