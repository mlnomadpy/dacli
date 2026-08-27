---
id: 01M11NVQ94M242167T8S5BXCBF
kind: event
schema_version: 1
event_kind: commit
created: 2026-08-27T13:17:00Z
created_by: a-root
about: "[[t-01M0D3MKRKCHSX8P51HRDF0HQX]]"
origin: agent
applied: true
checksum: sha256:f47911291a8e1fcf4853d8d1a1454b3d4ed204a935ac4b942b324e096780365f
---
2aacbcf1 fix: parse separated logs tail values

Mutation: restoring tail to the global boolean set makes TestLogsTailAcceptsDocumentedSeparatedValue fail with --tail parsed as true.
role: root
