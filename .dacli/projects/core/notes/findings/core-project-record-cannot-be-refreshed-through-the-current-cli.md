---
id: f-core-project-record-cannot-be-refreshed-through-the-current-cli
kind: note
note_kind: finding
created: 2026-08-19T14:45:19Z
created_by: a-fixer-wcmm5q
about: "[[t-01M0AEG5QK6WDSJQBZTG7Z8JWW]]"
severity: major
---
# Core project record cannot be refreshed through the current CLI
Measured with dacli project show core: the shared record still says Every planned() stub implemented and Go (141 files), while this checkout has 337 Go files. project add/list/show/rm expose no edit operation, and the linked-worktree instructions prohibit directly editing the shared main-workspace project file. A project-update command or owner-side state edit is required to satisfy the project-record acceptance item.
