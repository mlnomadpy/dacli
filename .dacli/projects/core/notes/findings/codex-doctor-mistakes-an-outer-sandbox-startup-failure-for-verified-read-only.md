---
id: f-codex-doctor-mistakes-an-outer-sandbox-startup-failure-for-verified-read-only
kind: note
note_kind: finding
created: 2026-08-12T16:26:40Z
created_by: a-codex-loop-auditor-aaqaj2
about: "[[386]]"
severity: major
---
# Codex doctor mistakes an outer sandbox startup failure for verified read-only enforcement
Reproduced on installed codex-cli 0.147.0-alpha.6.5: 'codex sandbox -P :read-only -C <tmp> -- touch <target>' exits 71 before running touch and prints 'sandbox-exec: sandbox_apply: Operation not permitted'; the target is absent. Yet a fresh workspace's 'runtime doctor' exits 0 and persists/reports 'sandbox verified (local codex sandbox refused a write)'. internal/features/execution/execution.go:305-333 treats any probe error + absent target + codexSandboxDenied output as verification, and codexSandboxDenied at :358-365 accepts the generic outer failure marker 'operation not permitted'. Thus an inability to start Codex's sandbox is certified as proof that the declared read-only policy ran, allowing ro spawns on an unverified adapter.
