---
id: role-codex-process-architect
kind: role
created: 2026-08-12T14:27:00Z
created_by: a-root
name: codex-process-architect
version: v1
summary: finish task 375 with durable process-tree identity and no recycled-PGID fallback
scope: "[internal/features/execution/**, internal/procmon/**]"
grant: rw
runtime: codex-rw
model: gpt-5.6-sol
max_points: 13
---
# codex-process-architect

Finish task 375 without weakening the task-369 guarantee that dacli never
signals an unrelated recycled process group.

## Method

Read `AGENTS.md`, `CONTRIBUTING.md`, task 375, and every finding about task 375
before editing. Start from the committed red task-177 regression on the task
branch.

The following fallback is forbidden because it is unsafe:
`recorded PID absent && GroupAlive(recorded PGID) => authenticated`. A numeric
group can be recycled; its new leader can fork a helper and exit, leaving the
unrelated group live with no process at the recorded PID. Rechecking the absent
PID does not distinguish that state from a genuine task-177 descendant.

Implement a real durable identity. Prefer a small persistent guardian/sentinel
that is the spawned process-group leader, whose PID and start identity are
recorded, and which remains alive until the runtime and its descendants drain.
The runtime must join that guardian's group. Liveness and kill paths may trust
the group only while the recorded guardian identity still matches. An
equivalent design is acceptable only if it distinguishes the counterexample
above after both the original and recycled leaders have exited.

Cover foreground and detached launch behavior, timeout/interrupt cleanup,
legacy records, Unix process groups, and Windows compilation. Preserve feature
slice boundaries. Prove the task-177 test red before the production fix and
keep the task-369 liveness and no-signal controls green. Use `dacli commit`, and
run formatting, vet, lint, and the full race suite before reporting completion.
