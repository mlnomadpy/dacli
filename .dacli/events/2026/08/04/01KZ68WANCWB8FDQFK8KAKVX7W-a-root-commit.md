---
id: 01KZ68WANCWB8FDQFK8KAKVX7W
kind: event
event_kind: commit
created: 2026-08-04T11:34:56Z
created_by: a-root
origin: agent
applied: true
---
2c96fe3 reconcile the backlog with what actually shipped, and file the reason it drifted

Thirteen tasks whose code has been merged on main for hours were still
sitting in open/: 205, 206, 210, 215, 222, 238, 241, 242, 243, 247, 248,
253, 255, plus the merge-wave task. `dacli next` was ranking work that
shipped this morning, and ranking it as must.

Each is closed through `accept --force --verify`, so every close names
the command that proves it rather than asserting it — the package suite
for the slice the task touched, and -race -count=2 for the seq-lock one.

257 is why it happened. The implementers committed and pushed but never
ran `task check`/`task done`, and `integrate --tasks <ref>` merges a
named branch without consulting the task's status, so nothing on the
landing path noticed. The gate exists — integrate over a project merges
DONE tasks — but naming tasks explicitly walks straight past it.

Also renumbers this session's second 256 to 257. That is the fourth
cross-branch seq collision today; 251 remains the fix.
role: root
