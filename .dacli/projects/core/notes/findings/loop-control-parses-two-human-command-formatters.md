---
id: f-loop-control-parses-two-human-command-formatters
kind: note
note_kind: finding
created: 2026-08-12T19:45:34Z
created_by: a-codex-maintainer-xscvft
about: "[[366]]"
severity: major
---
# Loop control parses two human command formatters
internal/features/orchestration/orchestration.go:2235 extracts a run ID via runIDFrom from spawn prose; internal/features/ship/ship.go:244 calls integratedCount, which scans integrate's user-facing summary. Changing either formatter changes orchestration state/diagnostics.
