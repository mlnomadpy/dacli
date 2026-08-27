---
id: t-01M11HZWC4XH41GVP71EJS8Z4V
kind: task
created: 2026-08-27T12:09:22Z
created_by: a-root
owner: a-root
github:
  issue: 811
  repo: mlnomadpy/dacli
---
# [agent-report] Stale transferred root claim blocks later task commits
## Context
Adopted from GitHub issue #811.

After a prior task transferred two file claims to a-root, claiming a new task and creating a new dacli worktree did not replace or extend that claim. dacli commit --task <new-task> refused every correctly scoped changed file as outside the old websocket-only claim. task claim exposes no file-scope repair command, so the only available workaround was dacli commit --force. Expected: a new claimed task/worktree should establish its own inferred claim, or dacli should expose an audited claim update command.

---
_Reported via `dacli report`._
- dacli: dev
- platform: darwin/arm64
- workspace and run transcript withheld (public upstream) — re-run with --disclose to include them

## Acceptance
## Log
