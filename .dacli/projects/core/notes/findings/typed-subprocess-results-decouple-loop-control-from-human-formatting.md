---
id: f-typed-subprocess-results-decouple-loop-control-from-human-formatting
kind: note
note_kind: finding
created: 2026-08-12T19:53:37Z
created_by: a-codex-maintainer-xscvft
about: "[[366]]"
severity: major
---
# Typed subprocess results decouple loop control from human formatting
internal/commandresult/commandresult.go carries Spawn.RunID and Integration.Merged through DACLI_COMMAND_RESULT; internal/features/orchestration/orchestration.go consumes Spawn instead of banner text, and internal/features/ship/ship.go consumes Integration instead of prose. Formatter-change regressions are in driver_test.go and ship_test.go.
