---
id: t-01M11HZW8X678270CFWDBK7YTN
kind: task
created: 2026-08-27T12:09:21Z
created_by: a-root
owner: a-root
github:
  issue: 813
  repo: mlnomadpy/dacli
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
---
# [agent-report] integrate skips a second PR for an active task after an earlier PR landed
## Context
Adopted from GitHub issue #813.

An active task intentionally used multiple bounded PR slices. PR #89 landed first while the parent task remained open. A fresh worktree/branch then produced fully green PR #90 for the same active task. dacli integrate --tasks <task> --pr --into dev --merge --force reported already landed using PR #89, ignored open PR #90, and cleaned the local worktree. The workaround was gh pr merge 90 --merge. Expected: integration status should resolve the current branch/open PR, not treat any historical landed PR for an active task as proof that all later task work landed.

---
_Reported via `dacli report`._
- dacli: dev
- platform: darwin/arm64
- workspace and run transcript withheld (public upstream) — re-run with --disclose to include them

## Acceptance
## Log
