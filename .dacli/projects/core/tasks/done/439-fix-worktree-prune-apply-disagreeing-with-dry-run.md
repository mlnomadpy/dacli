---
id: t-01KZYNVFR96911X4D1V3D9T98Y
kind: task
created: 2026-08-13T23:03:27Z
created_by: a-root
owner: a-root
estimate: "{optimistic: 1, probable: 2, pessimistic: 3}"
github:
  issue: 647
  repo: mlnomadpy/dacli
---
# Fix worktree prune apply disagreeing with dry-run
## Acceptance
- [x] worktree prune removes every clean worktree that its immediately preceding dry-run reports reclaimable
- [x] merged-branch and finished-run eligibility use the same predicate in preview and apply modes
- [x] a regression test reproduces dry-run reporting reclaimable followed by apply reclaiming zero, and proves the worktree registration and directory are removed
## Log
- 2026-08-13T23:04:17Z claimed by a-fixer-7w7rgg
- 2026-08-13T23:21:14Z accepted by a-root (applied 1 proposal(s))
- 2026-08-13T23:21:14Z verified by `go test ./internal/cli ./internal/gitx` (exit 0) in branch main at 9942ad3 — proves that tree builds, not that the work is in trunk
- 2026-08-13T23:21:14Z deliverable: dacli/439-fix-worktree-prune-apply-disagreeing-with-dry-run is merged into main
- 2026-08-13T23:21:14Z completed by a-root
- 2026-08-13T23:51:31Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/648 (event 01KZYPF8YV17T6HXA3B9RY485Y)
## Verification Evidence
{"command":"go test ./internal/cli ./internal/gitx","exit_code":0,"duration_ms":35990,"artifact_hash":"sha256:bea2769a6c3596a660e4609295c856278c37da47d9ff13b7f721aebccbb8ae55","verifier":"a-root","branch":"main","commit_sha":"9942ad30813e7361e2657453ce1a29e822bc4602"}
