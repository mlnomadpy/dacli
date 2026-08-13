---
id: f-runtime-execution-now-enforces-persisted-provider-cooldowns
kind: note
note_kind: finding
created: 2026-08-13T10:58:29Z
created_by: a-fixer-97fz9k
about: "[[406]]"
severity: major
trust: refuted
---
# Runtime execution now enforces persisted provider cooldowns
internal/features/execution/execution.go:555 checks the persisted breaker before launch, selects only an explicit role fallback, and records classified nonzero spawn/supervise outcomes through the shared runtime-limit store; internal/features/execution/providerlimits_test.go proves real spawn classification and permanent/policy non-fallback behavior.
