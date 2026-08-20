---
id: d-answer-may-this-task-s-claim-be-expanded-to-include-internal-cli-verification
kind: note
note_kind: decision
created: 2026-08-19T13:30:59Z
created_by: a-root
about: "[[t-01M0AGCX64GETGKKBBJK5SKG7D]]"
github:
  issue: 739
  repo: mlnomadpy/dacli
---
# Answer: May this task's claim be expanded to include internal/cli/verification_worktree_test.go so the required public-command linked-worktree regression can land with commit a26f04d? The commit gate refused that path; the scoped implementation and store non-Git test are committed.
## Chose
Q (a-fixer-z2ed21): May this task's claim be expanded to include internal/cli/verification_worktree_test.go so the required public-command linked-worktree regression can land with commit a26f04d? The commit gate refused that path; the scoped implementation and store non-Git test are committed.

A: Approved: expand the correction claim to internal/cli/verification_worktree_test.go together with the task 472 acceptance/planning/store paths. Preserve commit a26f04d, add the public-command linked-worktree regression, prove its mutation, and do not touch internal/gitx while task 471 owns it.
## Rejected
Force the original out-of-claim commit or omit the public-command regression
## Because
The public regression is an acceptance requirement, while exact path claims preserve isolation from the active task 471 work.
