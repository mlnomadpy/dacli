---
id: 01M0F66Q9F1MWWKG54XS5H9NCB
kind: event
schema_version: 1
event_kind: commit
created: 2026-08-20T08:57:03Z
created_by: a-root
about: "[[t-01M0CZANAQKP50AWEN2C6C8VXR]]"
origin: agent
applied: true
checksum: sha256:014ebe929d944dbd9823afb34483469753006300d618d33e760d43f7380e086d
---
0f5dc29 t-01M0CZANAQKP50AWEN2C6C8VXR: add audited dependency edits

Validate the complete typed graph before writing, route non-owner edits through a replay-safe dependency event, preserve stable IDs and adopted GitHub mappings, and update the dacli skill to use the shipped command.

Mutation: disabling the ComputeCPM error guard made TestDependencyChangeValidationFailuresDoNotWrite/cycle fail at dependency_test.go:80 with 'invalid dependency change succeeded'.
role: root
