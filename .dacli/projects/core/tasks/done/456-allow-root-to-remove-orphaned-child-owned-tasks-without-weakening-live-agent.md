---
id: t-01KZZR4CR10XX2BAZG1Y1ZDDZ7
kind: task
created: 2026-08-14T09:02:30Z
created_by: a-root
owner: a-root
github:
  issue: 667
  repo: mlnomadpy/dacli
priority: must
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
---
# Allow root to remove orphaned child-owned tasks without weakening live-agent safety
## Context
Adopted from GitHub issue #667.

Implementation and claim boundary: `internal/features/planning` owns `task rm` authorization, command behavior, and regression fixtures; consume existing identity/agent lifecycle state through `internal/agentid` or `internal/agentstate` only if the live-versus-retired predicate cannot be expressed in the planning slice. Preserve `store.RemoveTask` as the sole deletion primitive. Do not change generic `CanMutate` semantics for unrelated commands.

## Reproduction

Cycle 95 review created duplicate task 454, owned by its now-retired child agent. Acting as a-root with rw grant:

    dacli task rm 454
    dacli task rm 454 --force

Both return exit 3:

    454-... is owned by a-codex-loop-auditor-ejgvrk — only its owner or root can remove it

dacli whoami reports a-root (grant: rw, role: root). task claim 454 only records a proposal because root is not the current owner; sync cannot apply it because the object is child-owned.

## Proven cause

internal/features/planning/planning.go cmdTaskRm gates removal with id.CanMutate(t.Owner()). internal/agentid.CanMutate intentionally permits only an rw identity that owns the object and contains no root exception. The command message promises a root recovery path that the predicate cannot authorize.

This is distinct from closed #433. That issue protected tasks owned by actively working agents. Here the owner is retired, there is no live process, and the task is a proven duplicate that should never have existed.

## Manual workaround

None through dacli. Manual record-file deletion would bypass reference/index invariants and is not acceptable.

## Design

Keep the active-agent protection from #433. Add an explicit root-only orphan recovery path in task rm: resolve the owner lifecycle, refuse while that owner has a live run/process, and permit root to remove a child-owned task only when the owner is retired/non-live. Require --force when the task has history or is active so the audit intent is explicit.

## Acceptance
- [x] A regression creates a child-owned task, retires the child, and proves a-root can remove it through task rm with the documented explicit force when required.
- [x] The same command refuses while the child has a live run or process and names that live ownership as the reason.
- [x] A non-root sibling still cannot remove another agent owned task.
- [x] A read-only root identity cannot remove tasks.
- [x] Referenced-task and done-task safety checks remain enforced after authorization.
- [x] The success path uses store.RemoveTask so indexes, tombstones, and reference checks remain canonical.
- [x] Command help and refusal text describe the exact owner/root-orphan policy without promising an unreachable path.
- [x] Mutation evidence, focused planning/agent tests, and go test ./... pass.
## Log
- 2026-08-14T09:03:55Z claimed by a-maintainer-3gxynh
- 2026-08-14T09:23:25Z accepted by a-root
- 2026-08-14T09:23:25Z verified by `GOCACHE=/tmp/dacli-accept-456 go test ./...` (exit 0) in branch main at 1ef6fa3 — proves that tree builds, not that the work is in trunk
- 2026-08-14T09:23:25Z deliverable: dacli/456-allow-root-to-remove-orphaned-child-owned-tasks-without-weakening-live-agent is merged into main
- 2026-08-14T09:23:25Z completed by a-root
- 2026-08-14T09:25:41Z accepted by a-root
- 2026-08-14T09:25:41Z closed WITHOUT verification — no --verify command was given
- 2026-08-14T09:25:41Z deliverable: dacli/456-allow-root-to-remove-orphaned-child-owned-tasks-without-weakening-live-agent is merged into main
- 2026-08-14T09:25:41Z completed by a-root
- 2026-08-14T10:19:45Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/669 (event 01KZZRW0WM4CYZZXT2AASMRHK9)
## Verification Evidence
{"command":"GOCACHE=/tmp/dacli-accept-456 go test ./...","exit_code":0,"duration_ms":59125,"artifact_hash":"sha256:0774c259d7a75e9d209a4cf9ea8c1e385d64d93a8284390508ec456d3962ee66","verifier":"a-root","branch":"main","commit_sha":"1ef6fa3da72c1f5a19f2cf3eddb8e7db61dd03de"}
