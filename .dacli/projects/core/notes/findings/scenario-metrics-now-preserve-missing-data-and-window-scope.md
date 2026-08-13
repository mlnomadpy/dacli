---
id: f-scenario-metrics-now-preserve-missing-data-and-window-scope
kind: note
note_kind: finding
created: 2026-08-13T21:27:53Z
created_by: a-fixer-fwr9f3
about: "[[433]]"
severity: major
---
# Scenario metrics now preserve missing data and window scope
internal/metrics/metrics.go exposes schema_version 1 with explicit samples and nullable values; collector_test.go proves a named window excludes an older failed run while retaining output_tokens, max_tokens, and failure class from the selected run. Removing failure aggregation makes TestCompareNamedScenarioWindowsRejectsMissingOrFabricatedData fail at metrics_test.go:29.
