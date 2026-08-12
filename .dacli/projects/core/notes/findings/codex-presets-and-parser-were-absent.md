---
id: f-codex-presets-and-parser-were-absent
kind: note
note_kind: finding
created: 2026-08-12T15:58:01Z
created_by: a-codex-maintainer-vc0pbd
about: "[[371]]"
severity: major
---
# Codex presets and parser were absent
Focused regression fails to compile at internal/features/execution/stream_test.go:31: undefined teeStructuredJSON; existing presets in execution.go include only Claude/generic and execRuntime applies the Claude parser to the sole structured format.
