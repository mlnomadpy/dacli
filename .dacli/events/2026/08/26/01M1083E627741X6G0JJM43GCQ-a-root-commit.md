---
id: 01M1083E627741X6G0JJM43GCQ
kind: event
schema_version: 1
event_kind: commit
created: 2026-08-26T23:57:18Z
created_by: a-root
about: "[[t-01M0ZCAQ33YAXPS79D8EJ676KP]]"
origin: agent
applied: true
checksum: sha256:64adb001136119bf8db39c66425d73345514f18cc4375c3e8934e889d5f28e6b
---
548e112 fix: resolve PR base from project and repository policy

Mutation: forcing main makes TestPRUsesLinkedRepositoryDefaultMaster fail because pr create receives --base main.
role: root
