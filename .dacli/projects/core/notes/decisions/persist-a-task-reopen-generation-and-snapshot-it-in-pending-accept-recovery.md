---
id: d-persist-a-task-reopen-generation-and-snapshot-it-in-pending-accept-recovery
kind: note
note_kind: decision
created: 2026-08-17T15:32:01Z
created_by: a-maintainer-z795wm
about: "[[t-01M05Y78XTYNQ4GP3AV396JFD6]]"
---
# Persist a task reopen generation and snapshot it in pending-accept recovery
## Chose
Persist a task reopen generation and snapshot it in pending-accept recovery
## Rejected
Infer deliberate reopen from open status or only remove the journal entry in the planning command
## Because
Open also represents legitimate post-merge verifier recovery, while a durable task generation survives bounded-loop restarts and lets both orchestration and shared worktree pruning distinguish corrective work from the earlier landing
