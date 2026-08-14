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
- [ ] A regression creates a child-owned task, retires the child, and proves a-root can remove it through task rm with the documented explicit force when required.
- [ ] The same command refuses while the child has a live run or process and names that live ownership as the reason.
- [ ] A non-root sibling still cannot remove another agent owned task.
- [ ] A read-only root identity cannot remove tasks.
- [ ] Referenced-task and done-task safety checks remain enforced after authorization.
- [ ] The success path uses store.RemoveTask so indexes, tombstones, and reference checks remain canonical.
- [ ] Command help and refusal text describe the exact owner/root-orphan policy without promising an unreachable path.
- [ ] Mutation evidence, focused planning/agent tests, and go test ./... pass.
## Log
- 2026-08-14T09:03:55Z claimed by a-maintainer-3gxynh
