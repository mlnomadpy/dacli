---
id: 01M12VWAJA482ZQ47PKABB0K4G
kind: event
schema_version: 1
event_kind: commit
created: 2026-08-28T00:21:25Z
created_by: a-maintainer-z0nyk9
about: "[[t-01M1068M8HJ9G8XCXMEMVE2V8D]]"
origin: agent
applied: true
checksum: sha256:783e1fb55b9496c9320501daa58bd18e6ee9ae1db65263eb1f6340b82299f849
---
836579ef t-01M1068M8HJ9G8XCXMEMVE2V8D: scope dependency validation to changed graph

Resolve stored dependency shorthand in each edge owner project and validate only the component reachable from the changed task, so unrelated legacy ambiguity cannot fence future edits.

Mutation evidence before the fix:
--- FAIL: TestDependencyChangeIgnoresUnrelatedAmbiguousLegacyRefs
    dependency_test.go:76: project-qualified edit was blocked by unrelated legacy ref: task 002-q-legacy dependency "001": ref "001" is ambiguous: p/001-p-base, q/001-q-base
role: maintainer
