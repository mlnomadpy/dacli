---
id: f-event-replay-now-detects-payload-corruption-and-resumes-from-durable-per-event
kind: note
note_kind: finding
created: 2026-08-13T20:07:43Z
created_by: a-codex-maintainer-zkfgn1
about: "[[430]]"
severity: major
---
# Event replay now detects payload corruption and resumes from durable per-event checkpoints
internal/eventlog/eventlog.go writes schema_version 1 plus SHA-256 for immutable fields, validates it before listing or marking applied, and accepts legacy records with neither field. internal/eventlog/eventlog_test.go covers migration and mismatch; sync_test.go injects interruption after a persisted applied marker and proves restart applies each comment once. Mutation proof with checksum guard disabled failed: TestListRejectsChecksumMismatch reported events=1 holes=[]. Full go test ./... and go vet ./... pass; golangci-lint is unavailable in PATH.
