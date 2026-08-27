---
id: d-prefer-an-open-pr-on-a-reopened-task-s-canonical-branch-before-its-recorded
kind: note
note_kind: decision
created: 2026-08-27T22:06:48Z
created_by: a-fixer-zpvnda
about: "[[t-01M12K8SEVWQQJXS5MBPMTJWNR]]"
---
# Prefer an open PR on a reopened task's canonical branch before its recorded historical PR
## Chose
Prefer an open PR on a reopened task's canonical branch before its recorded historical PR
## Rejected
Always resolve the newest logged PR URL first
## Because
Reopen preserves earlier-generation logs while the branch is reused, so that URL can truthfully name a merged PR that is unrelated to the active correction.
