---
id: 01M0Z674ZKGC668JKGNG7QJ3A3
kind: event
schema_version: 1
event_kind: commit
created: 2026-08-26T14:05:08Z
created_by: a-fixer-qhwq3c
about: "[[t-01M0NR4J956431ZNW3MBDKCH0H]]"
origin: agent
applied: true
checksum: sha256:20c25d24a687b2f3d582c8b5e1cbbf4310366b3514e0f5177e82766405c4c745
---
b841352 t-01M0NR4J956431ZNW3MBDKCH0H: isolate Codex readiness from stderr flood

Keep stdout JSONL readiness and capped stderr diagnostics on separate drain paths, so warning floods cannot delay turn.started classification. Strengthen the hanging-tree fixture to exceed the diagnostic cap while fragmenting readiness.

Mutation: dropping readiness delivery makes TestCodexBehavioralPreflightReadinessStopsAndReapsHangingTree fail at preflight_test.go:361 with transient/transport behavioral launch readiness exceeded bounded deadline.
role: fixer
