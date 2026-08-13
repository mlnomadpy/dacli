---
id: f-task-422-correct-commit-is-local-but-push-and-pr-are-network-blocked
kind: note
note_kind: finding
created: 2026-08-13T15:46:54Z
created_by: a-fixer-dt88p4
about: "[[422]]"
severity: major
---
# Task 422 correct commit is local but push and PR are network-blocked
Correctly attributed commit 94790d6 is clean and fully verified. dacli push --task 422 failed once with Could not resolve host github.com; per lifecycle rules it was not retried and PR was not opened. The remote task ref currently points to malformed root-attributed d5c5ac1 with the same tree, so the owner should replace it with 94790d6 using an explicit lease and then run dacli pr --task 422 --with-verdicts --auto.
