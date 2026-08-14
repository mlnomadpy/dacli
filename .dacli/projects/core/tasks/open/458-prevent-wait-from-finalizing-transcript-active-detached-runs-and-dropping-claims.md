---
id: t-01KZZVFWZWP3M2KX52E1FF6CMA
kind: task
created: 2026-08-14T10:01:13Z
created_by: a-root
owner: a-root
priority: must
github:
  issue: 672
  repo: mlnomadpy/dacli
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
---
# Prevent wait from finalizing transcript-active detached runs and dropping claims
## Context
Adopted from GitHub issue #672.

Implementation and claim boundary: `internal/features/execution` owns spawn, agents, wait, recovery, and claim lookup orchestration; `internal/procmon` may change only if the process-identity primitive itself is proven wrong. Do not modify event disposition, collaboration, model, or store code for this lifecycle defect.

## Symptom

During task 457 / PR #671 on macOS, dacli finalized detached Codex workers while their transcripts and file edits were still advancing.

Concrete reproduction from 2026-08-14:

1. Spawned run `01KZZTGX07VHDQ9SEFHSKKQ4V4` with a detached Codex maintainer, isolated worktree, and an explicit multi-path claim for task 457.
2. Roughly one minute later, `dacli agents --tail` printed `no live agents`.
3. `dacli wait 01KZZTGX07` immediately recorded `no visible result`, acceptance 0/11.
4. After that terminal report, the same transcript continued through items 11-26, edited files, passed `go test ./...`, and committed `582bb60`.
5. Because the run had been prematurely recovered/finalized, `dacli commit` warned `no recorded --claim for a-maintainer-pxs3be — committing without scope enforcement`, despite the explicit spawn claim.

The same false-negative happened on corrective run `01KZZV57GGYJH5SRTFC8DMYZV9`: status said no live agents while its transcript continued running tests and later committed `edfe123`.

## Expected

- A detached run whose runtime process/transcript is still active remains live.
- `wait <run>` does not write a terminal outcome while transcript activity continues within the configured timeout.
- Claim records remain available until the actual process terminates, so `dacli commit` enforces the explicit spawn claim.

## Suspected cause

This appears to be a regression of #477 / task 382 and related #614 / task 436. When process-table visibility is restricted or the recorded leader cannot be observed, `wait` still treats the negative process probe as authoritative even though transcript activity is fresh. The recovery path then completes `proc.txt`, and claim lookup no longer sees the declared paths.

## Manual workaround used

Ignored the premature `no visible result`, tailed the run transcript directly, and avoided touching the worktree until transcript activity stopped. Spawned a new governed child to obtain an attributed commit after each recovery.

## Acceptance
- [ ] A regression launches or fixtures a detached run with an unobservable process identity and a transcript that continues advancing; `agents` and `wait` do not finalize it while activity is fresh and before timeout.
- [ ] The run's explicit `--claim` remains resolvable by `dacli commit` until real process termination.
- [ ] Once the process actually exits (or timeout is reached), `wait` finalizes exactly once and releases the claim.
- [ ] The test covers the observed sequence: status probe, named wait, continued transcript write, claim lookup.
- [ ] Mutation evidence, focused execution/procmon tests, and `go test ./...` pass.
## Log
- 2026-08-14T10:11:34Z claimed by a-maintainer-j68p78
