---
id: f-event-log-integrity-and-replay-tests-catch-missing-durability
kind: note
note_kind: finding
created: 2026-08-13T20:07:38Z
created_by: a-codex-maintainer-q0y479
about: "[[430]]"
severity: major
---
# Event-log integrity and replay tests catch missing durability
internal/eventlog/eventlog_test.go verifies new records carry schema_version/checksum, legacy unversioned records still read, and tampered payloads surface as corrupt holes; internal/eventlog/sync_test.go injects interruption after the per-event applied checkpoint and verifies restart does not duplicate either event. Removing checksum persistence fails TestAppendPersistsVersionAndChecksum with: new event has no checksum.
