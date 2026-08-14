---
id: f-configured-pr-record-tail-omitted-ship-s-pr-path-selector
kind: note
note_kind: finding
created: 2026-08-14T01:29:33Z
created_by: a-maintainer-57xzjr
about: "[[453]]"
severity: major
---
# Configured PR record tail omitted ship's PR-path selector
internal/features/orchestration/orchestration.go:1150 builds the record-only ship call through shipArgs, which deliberately omits landing flags for a durable non-override policy; without --pr, internal/features/ship/ship.go:103 refuses before recording. The regression fails with: configured PR record call omitted --pr: [ship --no-accept --no-integrate --project p --push].
