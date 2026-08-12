---
id: d-share-github-pr-state-classification-before-ancestry-fallback
kind: note
note_kind: decision
created: 2026-08-12T19:24:10Z
created_by: a-codex-maintainer-djpe71
about: "[[397]]"
github:
  issue: 530
  repo: mlnomadpy/dacli
---
# share GitHub PR-state classification before ancestry fallback
## Chose
share GitHub PR-state classification before ancestry fallback
## Rejected
infer squash landing from patch similarity or weaken ancestry checks
## Because
the task PR is the authoritative task-to-squash-commit link; similarity can falsely certify unrelated changes, while CLOSED and OPEN PRs must remain unlanded
