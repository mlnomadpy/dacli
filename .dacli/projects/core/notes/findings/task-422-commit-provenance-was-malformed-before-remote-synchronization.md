---
id: f-task-422-commit-provenance-was-malformed-before-remote-synchronization
kind: note
note_kind: finding
created: 2026-08-13T15:46:41Z
created_by: a-fixer-dt88p4
about: "[[422]]"
severity: major
---
# Task 422 commit provenance was malformed before remote synchronization
The first wrapper commit became d5c5ac1 with author a-root and subject -m, then origin/dacli/422 advanced to it. After soft-reset and the same documented wrapper with a one-line message, 94790d6 is correctly authored by a-fixer-dt88p4 with task trailers. dacli report could not file upstream because gh authentication is invalid.
