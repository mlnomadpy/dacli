---
id: t-01M1493JQ62NY4E39PQHNEZ6TP
kind: task
created: 2026-08-28T13:31:49Z
created_by: a-root
owner: a-root
github:
  issue: 872
  repo: mlnomadpy/dacli
estimate: "{optimistic: 8, probable: 13, pessimistic: 21}"
depends_on: "[t-01M147RA6982P2FQXN64RZFPG4, t-01M11HZW8X678270CFWDBK7YTN, t-01M146BA26TEH86YE16XHDZKGY]"
---
# Support typed delivery slices without a second task lifecycle
## Context
Adopted from GitHub issue #872.

## Parent

Extracted from real-project feedback in #871. Coordinate with aggregate/decomposition semantics in #866 and exact PR-generation reconciliation in #813/#858.

## Observed symptom

Long-running product tasks are often delivered through several independently reviewable PRs. Reusing one canonical task branch lets an older merged PR masquerade as proof that newer work landed; creating unrelated tasks loses the parent acceptance and progress relationship.

## Objective

Support typed delivery slices while preserving one task lifecycle. Prefer representing a slice as a typed child task/delivery generation—rather than inventing a second mutable ledger—when that can satisfy the invariants.

Suggested surface:

```bash
dacli slice add --task <ref> --title <title> --accept <criterion>
dacli pr --slice <task>/<slice>
dacli task progress <ref> --json
```



## Non-goals

- Encouraging oversized tasks instead of decomposition.
- Treating a slice as an untracked ad-hoc branch.
- Adding a second completion state machine that can disagree with tasks.

## Manual workaround today

Operators reuse a task branch or create loosely related child tasks, then manually track which PRs remain and prevent premature issue closure.

## Acceptance
- [ ] Every slice has a stable ID, parent task, generation, independent branch/head and tree SHA, acceptance evidence, PR identity, merge SHA, claims, and cleanup state.
- [ ] Two slices of one parent can be open or merged independently; a historical merged slice never proves a newer open slice landed.
- [ ] Parent progress is derived from required slices, and parent acceptance/issue closure refuses until every required slice is verified and freshly observed landed under project policy.
- [ ] Partial slice PRs reference the parent issue without closing it; only an explicitly terminal delivery may emit closing semantics.
- [ ] Critical path, `next`, aggregate progress, reconciliation, cleanup, GitHub projection, and JSON views agree on slice state.
- [ ] Crash/restart fixtures cover interruption after slice commit, PR creation, CI, merge, parent reconciliation, and cleanup without duplicate slices or PRs.
- [ ] Backward compatibility is explicit for projects that use one branch/PR per task.
- [ ] Mutation tests fail when an older merged slice satisfies a newer generation or closes the parent early.
## Log
- 2026-08-28T13:32:49Z dependency edit by a-root (event 01M1495D91HF76EGSKW7Z7YBR6)
