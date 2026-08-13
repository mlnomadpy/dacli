---
id: t-01KZYQ5EMEJQ52K002KXT47S38
kind: task
created: 2026-08-13T23:26:22Z
created_by: a-root
owner: a-root
github:
  issue: 628
  repo: mlnomadpy/dacli
---
# [agent-report] project-qualified task refs from ambiguity guidance do not resolve
## Context
Adopted from GitHub issue #628.

From the supabase task worktree, 'dacli task check 001 --n 1' reported ambiguity and suggested supabase/001-define-the-firebase-to-supabase-migration-contract-and-rollback-plan; using that exact ref returned not found. The globally unique bare slug resolves, then correctly enforces owner-only checking.

---
_Reported via `dacli report`._
- dacli: dev
- platform: darwin/arm64
- workspace and run transcript withheld (public upstream) — re-run with --disclose to include them

## Acceptance
## Log
