---
id: t-01KZVESETQE54M4AY46SEVF281
kind: task
created: 2026-08-12T17:02:17Z
created_by: a-codex-loop-auditor-hexawh
owner: a-root
priority: must
estimate: "{optimistic: 1, probable: 2, pessimistic: 3}"
github:
  issue: 491
  repo: mlnomadpy/dacli
---
# Fix estimate-task claim inference so required implementation slices are writable
## Acceptance
- [x] Given task 381's title and acceptance text, store.ClaimHints includes internal/store, internal/spm, internal/features/planning, internal/features/insight, and internal/cli
- [x] The loop's implementer spawn carries those inferred claims, and procmon.PathsOverlap confirms every file changed by the recorded task-381 implementation is covered
- [x] A regression based on task 381 fails when estimate/planning inference is removed and proves unrelated feature slices remain outside the claim
## Log
- 2026-08-12T18:23:28Z adopted by a-root (owner a-codex-loop-auditor-hexawh orphaned)
- 2026-08-12T19:02:38Z claimed by a-codex-maintainer-cr0hke
- 2026-08-12T19:19:34Z accepted by a-root (applied 1 proposal(s))
- 2026-08-12T19:19:34Z verified by `GOCACHE=/private/tmp/dacli-gocache GOTMPDIR=/private/tmp go test ./internal/store ./internal/features/orchestration` (exit 0) in branch main at ed41cb8 — proves that tree builds, not that the work is in trunk
- 2026-08-12T19:19:34Z deliverable: dacli/393-fix-estimate-task-claim-inference-so-required-implementation-slices-are-writable exists but is NOT in main — closed anyway
- 2026-08-12T19:19:34Z completed by a-root
- 2026-08-12T19:29:26Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/518 (event 01KZVP46DWWVD6G89BPK3D5JH9)
