---
id: f-task-436-is-locally-committed-but-remote-handoff-is-blocked
kind: note
note_kind: finding
created: 2026-08-13T21:06:42Z
created_by: a-codex-maintainer-8r5s5s
about: "[[436]]"
severity: major
---
# Task 436 is locally committed but remote handoff is blocked
Commit 4be744d is locally verified. push --task 436 failed resolving github.com and pr --task 436 --with-verdicts --auto failed connecting to api.github.com; no push, PR, auto-merge, acceptance, or landing is inferred. Acceptance checking is owner-gated to a-root.
