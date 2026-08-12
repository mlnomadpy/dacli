---
id: f-loop-wait-retained-a-finalized-codex-run-after-the-live-agent-view-was-empty
kind: note
note_kind: finding
created: 2026-08-12T13:16:10Z
created_by: a-root
about: "[[369]]"
severity: major
origin: internal/features/execution/execution.go:2418
---
# Loop wait retained a finalized Codex run after the live-agent view was empty
Reproduced in cycle 70 with run 01KZV1JNZSZQ9HVF26PBMMM8SQ. Codex left five dirty claimed files and exited with no final event after a focused test failure. 'dacli agents --tail' finalized it as no visible result and reported no live agents, yet the loop's already-started wait remained silent and blocked for more than two minutes until the operator interrupted it. proc.txt recorded pid/pgid 31183. This is concrete evidence for task 369's dead-leader/reused-group reconciliation acceptance; the run must never remain pending once the same liveness view has finalized it.
