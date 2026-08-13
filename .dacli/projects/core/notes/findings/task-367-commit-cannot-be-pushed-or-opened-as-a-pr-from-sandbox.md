---
id: f-task-367-commit-cannot-be-pushed-or-opened-as-a-pr-from-sandbox
kind: note
note_kind: finding
created: 2026-08-12T20:15:44Z
created_by: a-codex-maintainer-p44wb5
about: "[[367]]"
severity: major
---
# Task 367 commit cannot be pushed or opened as a PR from sandbox
Commit c8fb047 contains the verified implementation. Required github push dry-run failed because api.github.com was unreachable, so no public mutation was attempted and push/pr could not proceed. From a network-enabled context rerun dacli github push core 367 --dry-run, then dacli push --task 367 and dacli pr --task 367 --with-verdicts --auto.
