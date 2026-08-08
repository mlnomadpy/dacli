---
id: d-pr-first-integration-is-the-default-going-forward-local-merge-only-as-an
kind: note
note_kind: decision
created: 2026-07-22T21:32:26Z
created_by: a-root
---
# PR-first integration is the default going forward; local merge only as an offline/flaky-network fallback
## Chose
PR-first integration is the default going forward; local merge only as an offline/flaky-network fallback
## Rejected
keep merging task branches locally with git merge --no-ff
## Because
PRs give per-task tracking, review, issue linkage (Fixes #N), and CI — using GitHub fully, as the operator wants. dacli already has the pieces (spawn --pr, dacli pr with G3 enrichment + verify-verdicts-as-review-comments, pr --review). Local merge was a pragmatic choice for speed during rapid iteration + a flaky network; it stays only as the fallback when GitHub is unreachable
