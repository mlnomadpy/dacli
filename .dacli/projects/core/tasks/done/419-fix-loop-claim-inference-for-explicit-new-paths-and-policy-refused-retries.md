---
id: t-01KZXS3QH0F482KDNVNVVABKM1
kind: task
created: 2026-08-13T14:41:08Z
created_by: a-root
owner: a-root
priority: must
github:
  issue: 580
  repo: mlnomadpy/dacli
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
---
# Fix loop claim inference for explicit new paths and policy-refused retries
## Context
Adopted from GitHub issue #580.

Implementation belongs in `internal/store/store.go`, `internal/store/claimhints_test.go`, `internal/features/orchestration/orchestration.go`, and `internal/features/orchestration/driver_test.go`; keep claim derivation provider-neutral.

## Symptom

A real `dacli loop --project core --width 2 --max-cycles 8 --max-tokens 750000 --worker-timeout 1200 --into main --yolo` repeatedly spawned task 408 with the sole inferred claim `internal/features/execution`.

Task 408 explicitly requires creating `contracts/controlplane/v1`. The worker implemented and staged all 16 required files there, but `dacli commit` correctly refused with exit 3 because every changed file was outside the inferred claim. On the next cycle the loop re-picked task 408 unchanged. Its false live claim also made task 417's legitimate `internal/features/execution` spawn collide and be refused.

## Reproduction

1. Use task 408's current title/context/acceptance criteria.
2. Run a loop dry-run or live cycle.
3. Observe `spawn --claim internal/features/execution` instead of the requested `contracts/controlplane/v1`.
4. Stage a file under `contracts/controlplane/v1` and run `dacli commit`; it exits 3.
5. Let the next cycle begin; task 408 is selected again with the same claim.

## Suspected cause

`store.ClaimHints` only preserves literal paths that already exist. Because `contracts/controlplane/v1` is a new directory, it is dropped. The generic word `execution` in “source of truth for agent execution” then triggers the architectural vocabulary rule for `internal/features/execution`. A weak prose inference becomes the only hard write boundary.

This regresses the intent of #481/#385: an inferred hard claim must contain the required implementation, not merely match vocabulary.

## Manual recovery

I stopped the loop at its checkpoint, independently verified the staged task 408 implementation, and will recover its commit as the owner rather than retrying the identical exit-3 operation.

## Acceptance criteria

- [ ] A literal repository-relative path in task acceptance/context may describe a new path and remains eligible as a claim when its parent boundary is unambiguous.
- [ ] Generic prose such as “agent execution” does not infer `internal/features/execution` when the task explicitly names a different implementation boundary.
- [ ] Task 408 derives a claim that covers `contracts/controlplane/v1` and does not claim `internal/features/execution`.
- [ ] Claim collisions among a planned wave are detected before spawning, and non-conflicting tasks can still run.
- [ ] An exit-3 policy refusal caused by a claim mismatch is not immediately classified as “produced nothing” and re-picked unchanged.
- [ ] Dry-run and live planning derive identical claims.
- [ ] Regression tests make the task 408/task 417 collision fail before the fix and pass after it.

## Acceptance
- [x] A literal repository-relative path in task acceptance or context may describe a new path and remains eligible as a claim when its parent boundary is unambiguous
- [x] Generic prose such as agent execution does not infer internal/features/execution when the task explicitly names a different implementation boundary
- [x] Task 408 derives a claim covering contracts/controlplane/v1 without claiming internal/features/execution
- [x] Claim collisions among a planned wave are detected before spawning while non-conflicting tasks remain runnable
- [x] Exit-3 claim policy refusals are not classified as produced nothing and immediately re-picked unchanged
- [x] Dry-run and live planning derive identical claims
- [x] Regression tests reproduce the task 408 and task 417 collision before the fix and pass after it
## Log
- 2026-08-13T14:52:02Z claimed by a-fixer-dd0fvf
- 2026-08-13T15:10:59Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/584 (event 01KZXTA5M5N29SQ50T24HX86GF)
- 2026-08-13T15:11:27Z accepted by a-root (applied 1 proposal(s))
- 2026-08-13T15:11:27Z closed WITHOUT verification — no --verify command was given
- 2026-08-13T15:11:27Z deliverable: dacli/419-fix-loop-claim-inference-for-explicit-new-paths-and-policy-refused-retries is merged into main
- 2026-08-13T15:11:27Z completed by a-root
- 2026-08-13T15:20:25Z accepted by a-root
- 2026-08-13T15:20:25Z closed WITHOUT verification — no --verify command was given
- 2026-08-13T15:20:25Z deliverable: dacli/419-fix-loop-claim-inference-for-explicit-new-paths-and-policy-refused-retries is merged into main
- 2026-08-13T15:20:25Z completed by a-root
