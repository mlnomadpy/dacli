---
id: d-normalize-legacy-blocks-to-fs-at-task-creation-and-compatibility-reads
kind: note
note_kind: decision
created: 2026-08-27T21:53:40Z
created_by: a-fixer-dqsb6g
about: "[[t-01M12K8SH454ZH3Z1MB1Q3D4TG]]"
---
# Normalize legacy :blocks to FS at task creation and compatibility reads
## Chose
Normalize legacy :blocks to FS at task creation and compatibility reads
## Rejected
Reject every legacy alias and leave stored records unusable
## Because
FS preserves the historic blocking semantics while ensuring newly persisted dependencies use only supported types.
