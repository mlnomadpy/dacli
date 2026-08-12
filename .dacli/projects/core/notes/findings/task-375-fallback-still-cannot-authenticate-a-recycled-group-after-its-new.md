---
id: f-task-375-fallback-still-cannot-authenticate-a-recycled-group-after-its-new
kind: note
note_kind: finding
created: 2026-08-12T14:15:43Z
created_by: a-root
about: "[[375]]"
severity: critical
---
# Task 375 fallback still cannot authenticate a recycled group after its new leader exits
The proposed ReconcileRun returns true whenever the recorded PID is absent and GroupAlive(PGID) is true. Counterexample: the original group empties; the numeric PID/PGID is later reused by an unrelated leader with a different start identity; that new leader forks a helper and exits. At observation time the recorded PID is absent and the unrelated group remains live, exactly matching the task-177 shape, so ReconcileRun authenticates and terminateRecordedTree may signal the stranger. PIDStart cannot distinguish the two because neither leader remains observable. Acceptance requires durable descendant identity or an equivalent guardian/sentinel; the current PID/PGID/PIDStart record is insufficient.
