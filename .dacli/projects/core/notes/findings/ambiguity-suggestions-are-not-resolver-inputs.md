---
id: f-ambiguity-suggestions-are-not-resolver-inputs
kind: note
note_kind: finding
created: 2026-08-14T00:38:31Z
created_by: a-maintainer-1w0gkw
about: "[[445]]"
severity: major
---
# Ambiguity suggestions are not resolver inputs
internal/store/store.go:1271 emits project/NNN-slug suggestions, but matchesRef and TaskIndex only recognize bare keys; the regression fails with suggested ref core/001-dup: not found.
