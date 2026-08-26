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
- [x] In local landing mode, `spawn --review --task <ref>` resolves the canonical task branch or worktree without requiring a GitHub PR.
- [x] In PR landing mode, review resolution continues to use and validate the associated GitHub PR.
- [x] A local read-only reviewer receives the intended task diff/context and can record findings or a verdict without mutating product files.
- [x] Missing local branches, ambiguous task references, and genuine PR-mode failures return actionable non-success outcomes without creating a misleading successful review.
- [x] Tests cover local mode with no `gh` access, PR mode, detached review finalization, and task branches created in linked worktrees.
- [x] Mutation proof demonstrates restoring the unconditional PR lookup makes the local-mode test fail.
## Log
- 2026-08-26T13:14:07Z claimed by a-fixer-s3nggb
- 2026-08-26T13:35:22Z completed by a-root
- 2026-08-26T13:36:34Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/783 (event 01M0Z4405KDZBW1R0P59PW1VM3)
## Verification Evidence
{"command":"GOCACHE=/private/tmp/dacli-go-cache go test ./...","exit_code":0,"duration_ms":1634,"artifact_hash":"sha256:07db5a7f494d86162ce19dd327d134bb8ac313296b8ab5376f50a81125baa384","verifier":"a-root","branch":"dacli/481-agent-report-local-review-spawn-refuses-task-branch-without-a-github-pr","commit_sha":"a18e8e1068d240c18ce3d9ba0d8f02ed4b2fc6a8"}
