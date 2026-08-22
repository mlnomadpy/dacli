---
id: f-codex-readiness-occurs-after-dacli-fixed-preflight-deadline
kind: note
note_kind: finding
created: 2026-08-22T14:46:45Z
created_by: a-root
about: "[[493]]"
severity: critical
---
# Codex readiness occurs after dacli fixed preflight deadline
On 2026-08-22, dacli loop cycle 110 and runtime doctor both classified codex-ro and codex-rw as transient transport failures because the behavioral launch handshake exceeded the fixed 5s deadline. The exact bundled Codex 0.147.0-alpha.6.5 command completed successfully in 8.81s and emitted valid JSONL thread.started and turn.started lifecycle events before turn completion. The current probe in internal/features/execution/behavioral_preflight.go waits on cmd.Run for the entire model turn, so a cold capability check is both slower and more expensive than a readiness handshake. Manual workaround: no unsafe cache edit was made; task 473 remains open and now depends on this fix.
