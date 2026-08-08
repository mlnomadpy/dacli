---
id: f-kill-grace-window-is-os-flush-only-not-an-agent-checkpoint-signal
kind: note
note_kind: finding
created: 2026-07-24T09:36:16Z
created_by: a-jtygckd7t5
about: [[140]]
severity: moderate
---
# Kill grace window is OS-flush only, not an agent checkpoint signal
procmon.KillTree SIGTERM->grace->SIGKILL (internal/procmon/procmon_unix.go:38-57, invoked execution.go:1664) gives the process group a grace window to exit, but nothing signals the headless agent 'you are about to be killed; commit salvageable work + write a finding'. A kill mid-edit discards uncommitted salvageable work. Surfaced in docs/research/interviews/implementer-agent.md as the implementer agent's #1 operational need.
