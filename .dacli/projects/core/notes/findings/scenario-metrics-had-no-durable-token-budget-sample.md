---
id: f-scenario-metrics-had-no-durable-token-budget-sample
kind: note
note_kind: finding
created: 2026-08-13T21:25:14Z
created_by: a-fixer-q7facv
about: "[[433]]"
severity: major
---
# Scenario metrics had no durable token-budget sample
internal/features/execution/execution.go previously did not write --max-tokens into invocation.txt, so a metrics collector could not distinguish an absent budget from zero; task 433 now records max_tokens for spawn and supervise runs.
