---
id: f-generated-roster-preamble-contradicted-the-shipped-rw-sandbox-gate
kind: note
note_kind: finding
created: 2026-08-13T13:33:03Z
created_by: a-fixer-xdgjvd
about: "[[414]]"
severity: major
---
# Generated roster preamble contradicted the shipped rw sandbox gate
internal/features/catalog/catalog.go:225 said rw runtime capability was not checked, while internal/features/execution/execution.go:1582 refuses an rw grant when an allowlist grants no Edit/Write; regenerating docs/ROSTER.md faithfully reproduced the stale claim.
