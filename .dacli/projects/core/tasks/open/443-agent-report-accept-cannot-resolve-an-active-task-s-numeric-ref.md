---
id: t-01KZYQ5EF1YQ05NRA8GW3N9PQM
kind: task
created: 2026-08-13T23:26:22Z
created_by: a-root
owner: a-root
github:
  issue: 636
  repo: mlnomadpy/dacli
---
# [agent-report] accept cannot resolve an active task's numeric ref
## Context
Adopted from GitHub issue #636.

From task worktree dacli/001-define-the-firebase-to-supabase-migration-contract-and-rollback-plan, after a provenance commit, ran dacli accept 001 --verify with an explicit worktree cd. It returned ref 001 is ambiguous, listing supabase/001 among others. The task brief prescribes dacli accept 001 and says not to retry refused operations, leaving no accepted task state.

---
_Reported via `dacli report`._
- dacli: dev
- platform: darwin/arm64
- workspace and run transcript withheld (public upstream) — re-run with --disclose to include them

## Acceptance
## Log
