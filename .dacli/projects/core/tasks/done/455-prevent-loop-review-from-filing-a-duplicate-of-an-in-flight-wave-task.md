---
id: t-01KZZR4CN0HWN232ZD2GYGQDFP
kind: task
created: 2026-08-14T09:02:30Z
created_by: a-root
owner: a-root
github:
  issue: 668
  repo: mlnomadpy/dacli
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
---
# Prevent loop review from filing a duplicate of an in-flight wave task
## Context
Adopted from GitHub issue #668.

Implementation and claim boundary: `internal/store` owns non-mutating task similarity/duplicate detection and `internal/features/orchestration` owns the just-completed-wave review brief. Strengthen those two seams with contrasting regressions; do not add model-specific prompt exceptions or GitHub-only deduplication.

## Reproduction

During dacli loop cycle 95:

- Active task 443 / GitHub #636 was implementing stable ULIDs in generated mutating commands.
- Its Codex worker produced commit 80527aa changing internal/features/execution prompt construction and adding the cross-project sequence-001 regression.
- The review agent ran the required open and active task checks.
- It nevertheless created task 454, Fix generated worker prompts using ambiguous numeric refs for mutations, citing the same execution.go locations, the same alpha/001 and beta/001 reproduction, the same ULID workaround, and the same mutation test.
- task add accepted the duplicate instead of returning the documented near-duplicate refusal.

Task 454 is now an orphaned active record, and task 443 has merged via PR #665 and closed #636.

## Proven cause

The review prompt asks for semantic duplicate checking, but the local creation gate did not recognize the differently worded title even though the candidate So that and acceptance text overlap the active task Context and Acceptance almost exactly. The audit also reasoned from main before the in-flight branch landed and did not treat the just-completed wave commit as queued work.

## Manual workaround

The owner detected the duplication by comparing task 454 with active task 443 and commit 80527aa. Removal is separately blocked by #667.

## Design

Make loop review duplicate prevention structural, not solely a model instruction. Provide the review phase with the just-completed wave task IDs, statuses, branch commits, and linked issues. Before accepting a review-created task, compare its title plus problem/acceptance text against open, active, and pending-landing tasks. A candidate matching the same observable failure and implementation boundary must be refused with the existing task reference.

## Acceptance
- [x] A regression seeds active task 443 style text and attempts task 454 style wording; task creation is refused as a semantic duplicate and names the existing task.
- [x] Duplicate comparison includes open, active, and pending-landing tasks, not only ready/open titles.
- [x] Loop review briefs identify the just-completed wave task IDs, branch commits, linked GitHub issues, and whether each is pending PR landing.
- [x] The guard compares normalized problem and acceptance content in addition to titles without merging distinct defects that merely share a directory or generic words.
- [x] An honest empty review remains valid when every reproduced candidate is already represented.
- [x] The refusal is non-mutating: no task file, sequence allocation, ownership record, or GitHub mirror event is created.
- [x] A contrasting regression proves a genuinely different generated-reference defect can still be filed.
- [x] Mutation evidence, focused orchestration/store similarity tests, and go test ./... pass.
## Log
- 2026-08-16T17:41:18Z claimed by a-maintainer-n5gm5y
- 2026-08-16T18:00:53Z accepted by a-root
- 2026-08-16T18:00:53Z verified by `GOCACHE=/tmp/dacli-455-final go test ./...` (exit 0) in branch main at 706d648 — proves that tree builds, not that the work is in trunk
- 2026-08-16T18:00:53Z deliverable: dacli/455-prevent-loop-review-from-filing-a-duplicate-of-an-in-flight-wave-task is merged into main
- 2026-08-16T18:00:53Z completed by a-root
- 2026-08-16T18:06:25Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/677 (event 01M05V73WY5WSPTC8N20QW7GEN)
## Verification Evidence
{"command":"GOCACHE=/tmp/dacli-455-accept go test ./...","exit_code":0,"duration_ms":73898,"artifact_hash":"sha256:f51c01aba3bcc5cf2d9a8ddc0aba5395890519383a09f8d53610032a63827084","verifier":"a-root","branch":"main","commit_sha":"706d648876e564bbaa04143c01f47b1476291391"}
{"command":"GOCACHE=/tmp/dacli-455-final go test ./...","exit_code":0,"duration_ms":72591,"artifact_hash":"sha256:fda3f4e1c2b97255b826627d2568ed2fb7abc267e39a0e8d8ef8460af298f730","verifier":"a-root","branch":"main","commit_sha":"706d648876e564bbaa04143c01f47b1476291391"}
