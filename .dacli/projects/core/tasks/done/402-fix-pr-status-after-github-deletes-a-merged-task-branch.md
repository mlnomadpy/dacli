---
id: t-01KZX5XZ9VHSM4X1ST5JA2M9JA
kind: task
created: 2026-08-13T09:05:57Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 1, probable: 2, pessimistic: 3}"
github:
  issue: 545
  repo: mlnomadpy/dacli
---
# Fix PR status after GitHub deletes a merged task branch
## Acceptance
- [x] A task with a recorded PR URL remains merged after its remote head branch is deleted
- [x] PR status resolves the recorded PR before falling back to branch-head lookup or ancestry
- [x] Acceptance recognizes the merged PR without --allow-unlanded
- [x] Regression coverage reproduces squash merge followed by remote branch deletion
## Log
- 2026-08-13T09:28:00Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/546 (event 01KZX6DA5083TYQD7SR3W6EX5V)
- 2026-08-13T09:28:33Z accepted by a-root
- 2026-08-13T09:28:33Z verified by `GOCACHE=/private/tmp/dacli-go-cache go test ./internal/features/vcs ./internal/features/acceptance -run 'TestPRStatusUsesRecordedURLAfterMergedHeadDeletion|TestRecordedMergedPRReadsAsLandedAfterHeadDeletion' -count=1` (exit 0) in branch main at f244e11 — proves that tree builds, not that the work is in trunk
- 2026-08-13T09:28:33Z deliverable: dacli/402-fix-pr-status-after-github-deletes-a-merged-task-branch is merged into main
- 2026-08-13T09:28:33Z completed by a-root
