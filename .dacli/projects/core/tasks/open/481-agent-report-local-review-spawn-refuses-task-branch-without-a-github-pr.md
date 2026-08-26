---
id: t-01M0CZANK00P2B5XY6TVJNAWCK
kind: task
created: 2026-08-19T12:18:23Z
created_by: a-root
owner: a-root
github:
  issue: 714
  repo: mlnomadpy/dacli
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
---
# [agent-report] Local review spawn refuses task branch without a GitHub PR
## Context
Adopted from GitHub issue #714.

Task 010 was spawned with PR-first off and has canonical local task branch dacli/010-make-supabase-the-sole-runtime-and-remove-firebase. A read-only 'dacli spawn --task <full-id> --role supabase-reviewer --review --detach' run 01M0CYAK84 stopped after gh pr list failed, explicitly refusing to review the local branch. Expected: in local landing mode, --review resolves and checks the task branch/worktree without requiring GitHub. Actual: no visible result and zero events.

---
_Reported via `dacli report`._
- dacli: dev
- platform: darwin/arm64
- workspace and run transcript withheld (public upstream) — re-run with --disclose to include them

## Acceptance
- [ ] In local landing mode, `spawn --review --task <ref>` resolves the canonical task branch or worktree without requiring a GitHub PR.
- [ ] In PR landing mode, review resolution continues to use and validate the associated GitHub PR.
- [ ] A local read-only reviewer receives the intended task diff/context and can record findings or a verdict without mutating product files.
- [ ] Missing local branches, ambiguous task references, and genuine PR-mode failures return actionable non-success outcomes without creating a misleading successful review.
- [ ] Tests cover local mode with no `gh` access, PR mode, detached review finalization, and task branches created in linked worktrees.
- [ ] Mutation proof demonstrates restoring the unconditional PR lookup makes the local-mode test fail.
## Log
- 2026-08-26T13:14:07Z claimed by a-fixer-s3nggb
