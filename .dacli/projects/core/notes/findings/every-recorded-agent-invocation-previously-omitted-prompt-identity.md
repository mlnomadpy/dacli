---
id: f-every-recorded-agent-invocation-previously-omitted-prompt-identity
kind: note
note_kind: finding
created: 2026-08-19T14:29:23Z
created_by: a-maintainer-anf4d3
about: "[[t-01M0CX03CC0N95X4M5ESKRP2E6]]"
severity: major
---
# Every recorded agent invocation previously omitted prompt identity
internal/features/execution/execution.go:992 and verify.go:165 wrote runtime/task/model context but no prompt schema or content hash, so run records could not identify which mutable prompt contract was delivered.
