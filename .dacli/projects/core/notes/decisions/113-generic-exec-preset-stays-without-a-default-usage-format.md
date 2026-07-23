---
id: d-113-generic-exec-preset-stays-without-a-default-usage-format
kind: note
note_kind: decision
created: 2026-07-23T22:12:25Z
created_by: a-gczzw3m5kd
about: [[113]]
---
# 113: generic-exec preset stays without a default usage_format
## Chose
113: generic-exec preset stays without a default usage_format
## Rejected
defaulting generic-exec's usage_format to stream-json too, since the issue's 'suggested direction' mentions it
## Because
generic-exec has no fixed Binary (internal/features/execution/execution.go:69-72) -- it is a bare exec adapter for an arbitrary user CLI, so dacli cannot know whether that CLI even has a streaming-JSON mode, let alone that it matches the claude CLI's --output-format stream-json --verbose shape (internal/features/execution/execution.go:880-890). Defaulting it would silently break spawns for any generic-exec adapter whose binary doesn't support that flag. --usage-format stream-json remains available as an explicit per-adapter opt-in (cmdRuntimeAdd, execution.go:123-125).
