---
id: f-task-423-commit-verified-locally-but-push-and-pr-are-network-blocked
kind: note
note_kind: finding
created: 2026-08-13T15:58:30Z
created_by: a-fixer-f6typj
about: "[[423]]"
severity: major
---
# Task 423 commit verified locally but push and PR are network-blocked
Branch is clean at correctly attributed commit f174c05 after full verification. dacli push --task 423 failed once because github.com DNS could not resolve; per lifecycle rules it was not retried, and PR creation could not proceed. The remote tracking branch currently contains malformed root-attributed 9cf790e with the same patch tree, while local f174c05 carries correct fixer provenance; owner should replace it with an explicit lease, then run dacli pr --task 423 --with-verdicts --auto.
