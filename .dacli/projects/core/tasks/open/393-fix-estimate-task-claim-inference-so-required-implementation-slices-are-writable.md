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
- [ ] Given task 381's title and acceptance text, store.ClaimHints includes internal/store, internal/spm, internal/features/planning, internal/features/insight, and internal/cli
- [ ] The loop's implementer spawn carries those inferred claims, and procmon.PathsOverlap confirms every file changed by the recorded task-381 implementation is covered
- [ ] A regression based on task 381 fails when estimate/planning inference is removed and proves unrelated feature slices remain outside the claim
## Log
- 2026-08-12T18:23:28Z adopted by a-root (owner a-codex-loop-auditor-hexawh orphaned)
