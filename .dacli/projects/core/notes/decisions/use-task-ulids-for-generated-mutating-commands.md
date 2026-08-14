---
id: d-use-task-ulids-for-generated-mutating-commands
kind: note
note_kind: decision
created: 2026-08-14T01:48:01Z
created_by: a-maintainer-204p4w
about: "[[443]]"
---
# Use task ULIDs for generated mutating commands
## Chose
Use task ULIDs for generated mutating commands
## Rejected
Keep numeric prompt refs and rely on project qualification
## Because
worker instructions must remain stable across projects and task 445's qualified grammar is human shorthand rather than the machine identity
