---
id: t-01KZ704033W6SD9KF3W92487W7
kind: task
created: 2026-08-04T18:21:05Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 2, probable: 4, pessimistic: 8}"
---
# Ten object types can be created but have no removal command
## So that
a mistake made by a command is undoable by a command rather than by hand-editing markdown
## Acceptance
- [x] each creatable object type either gains a removal inverse or its absence is a documented decision
- [x] removal refuses when something still references the object, rather than leaving a dangling link
## Log
- 2026-08-09T23:08:08Z accepted by a-root
- 2026-08-09T23:08:08Z verified by `go test ./internal/store/...` (exit 0)
- 2026-08-09T23:08:08Z deliverable: no dacli/293-ten-object-types-can-be-created-but-have-no-removal-command branch — nothing to check against trunk
- 2026-08-09T23:08:08Z completed by a-root
