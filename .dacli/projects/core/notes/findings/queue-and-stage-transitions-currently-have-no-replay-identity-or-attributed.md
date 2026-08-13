---
id: f-queue-and-stage-transitions-currently-have-no-replay-identity-or-attributed
kind: note
note_kind: finding
created: 2026-08-13T19:57:23Z
created_by: a-codex-maintainer-2j651b
about: "[[431]]"
severity: major
---
# Queue and stage transitions currently have no replay identity or attributed transition journal
internal/features/queues/queues.go:98 advances or halts directly after ownership checks, and internal/features/stagegate/stagegate.go:137 advances directly after grant checks; neither accepts an idempotency key, persists terminal dead-letter state, nor appends an audit event. Repeating a successful queue advance moves the cursor again.
