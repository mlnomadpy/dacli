---
id: f-task-490-git-branch-name-parity-corrected
kind: note
note_kind: finding
created: 2026-08-26T15:17:16Z
created_by: a-root
about: "[[490]]"
severity: major
---
# Task 490 Git branch-name parity corrected
Second PR #795 re-review found HEAD accepted although git check-ref-format --branch HEAD rejects it. Added HEAD rejection plus a valid/invalid matrix executed against Git itself. The parity test also corrected an over-rejection: Git accepts @ under --branch. Mutation removing the HEAD guard fails TestLandingBaseValidationMatchesGitCheckRefFormat on HEAD; restored focused tests pass.
