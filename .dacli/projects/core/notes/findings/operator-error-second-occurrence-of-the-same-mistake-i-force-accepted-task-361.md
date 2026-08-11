---
id: f-operator-error-second-occurrence-of-the-same-mistake-i-force-accepted-task-361
kind: note
note_kind: finding
created: 2026-08-11T13:32:23Z
created_by: a-root
origin: internal/mcp/tools.go:66
---
# Operator error, second occurrence of the same mistake: I force-accepted task 361 immediately after writing a commit that said its work was deliberately NOT done ('filed as 361 rather than half-done'). Identical to the task 336 incident on 2026-08-10. Both times the trigger was the same: running 'accept --force' as a reflex at the end of a sprint, on a task list I had not re-read. The tool was honest both times — it stamped 'closed WITHOUT verification' — and both times I did not look. What is different now is the recovery: 'dacli task reopen 361 --reason ...' fixed it in one command and cleared the three boxes, where 336 needed hand-editing the store. That command exists BECAUSE of 336, and this is its second real use. The durable lesson is not about the tool: --force at the end of a sprint should be per-task with a stated reason, never a sweep.
