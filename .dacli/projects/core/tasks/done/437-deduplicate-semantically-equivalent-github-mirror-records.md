---
id: t-01KZYHZTP8NJVY9TYM7S2ANJ38
kind: task
created: 2026-08-13T21:55:55Z
created_by: a-root
owner: a-root
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
github:
  issue: 635
  repo: mlnomadpy/dacli
---
# Deduplicate semantically equivalent GitHub mirror records
## Acceptance
- [x] github push canonicalizes decisions and findings by normalized semantic content before planning remote writes
- [x] repeated recovery and network blocker records collapse to one GitHub comment or decision issue while distinct evidence remains visible
- [x] tests cover marker-idempotent replay, near-duplicate titles, and partial-push resume without suppressing a materially different record
## Log
- 2026-08-13T22:20:42Z claimed by a-fixer-sfwpk8
- 2026-08-13T23:02:15Z accepted by a-root (applied 1 proposal(s))
- 2026-08-13T23:02:15Z verified by `go test ./internal/features/ghmirror ./internal/cli` (exit 0) in branch main at 8eec229 — proves that tree builds, not that the work is in trunk
- 2026-08-13T23:02:15Z deliverable: dacli/437-deduplicate-semantically-equivalent-github-mirror-records is merged into main
- 2026-08-13T23:02:15Z completed by a-root
## Verification Evidence
{"command":"go test ./internal/features/ghmirror ./internal/cli","exit_code":0,"duration_ms":43398,"artifact_hash":"sha256:bb1ec31c6b27d23231aec0ee04cfa03bc328bf3e3a2d6e1e9707b5d85b1bd399","verifier":"a-root","branch":"main","commit_sha":"8eec2294529baf02e782fe04b173097228b0efa6"}
