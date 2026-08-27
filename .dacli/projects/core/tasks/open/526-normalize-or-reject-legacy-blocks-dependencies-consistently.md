---
id: t-01M12K8SH454ZH3Z1MB1Q3D4TG
kind: task
created: 2026-08-27T21:50:57Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
---
# Normalize or reject legacy blocks dependencies consistently
## Acceptance
- [ ] task add handles :blocks consistently with task depend: either normalize it to FS before persistence or reject it with FS/SS/FF/SF guidance.
- [ ] Existing stored :blocks aliases do not prevent unrelated dependency edits or critical-path analysis; behavior is covered by a migration or targeted compatibility regression.
- [ ] A round-trip regression proves no newly created task persists an unsupported dependency type.
- [ ] go test ./... passes.
## Log
- 2026-08-27T21:53:05Z claimed by a-fixer-dqsb6g
