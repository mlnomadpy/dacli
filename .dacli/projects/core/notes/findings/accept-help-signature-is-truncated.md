---
id: f-accept-help-signature-is-truncated
kind: note
note_kind: finding
created: 2026-08-19T12:14:46Z
created_by: a-maintainer-ebqr9f
about: "[[t-01M0CX031NDQ5PQ8VRX1PQNWXE]]"
severity: major
---
# Accept help signature is truncated
A freshly built CLI prints a truncated dacli accept usage ending after verify; internal/features/acceptance/acceptance.go around line 34 contains the malformed Usage string. The handler may accept the intended flags, but a context-free agent cannot validate the documented close command against help.
