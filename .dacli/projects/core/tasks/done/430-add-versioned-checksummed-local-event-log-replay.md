---
id: t-01KZYA0JJ09P6AEGHB6ST5HB2M
kind: task
created: 2026-08-13T19:36:31Z
created_by: a-codex-loop-auditor-hxqjcg
owner: a-root
priority: should
estimate: "{optimistic: 5, probable: 8, pessimistic: 13}"
parent: "[[t-01KZXXZPBXT332W00RP94HTR2K]]"
github:
  issue: 604
  repo: mlnomadpy/dacli
---
# Add versioned checksummed local event-log replay
## So that
corrupt or interrupted local records are detectable and old logs remain recoverable
## Acceptance
- [x] internal/eventlog persists a schema version and checksum with every new event while continuing to read the current unversioned format
- [x] replay checkpoints resume without duplicating already-applied events after an injected interruption
- [x] migration and corruption tests cover an old log, a checksum mismatch, and a restart from a checkpoint
## Log
- 2026-08-13T19:56:08Z claimed by a-codex-maintainer-6f32zj
- 2026-08-13T20:22:20Z adopted by a-root (owner a-codex-loop-auditor-hxqjcg orphaned)
- 2026-08-13T20:22:20Z accepted by a-root
- 2026-08-13T20:22:20Z verified by `go test ./internal/eventlog` (exit 0) in branch main at ad839af — proves that tree builds, not that the work is in trunk
- 2026-08-13T20:22:20Z deliverable: dacli/430-add-versioned-checksummed-local-event-log-replay is merged into main
- 2026-08-13T20:22:20Z completed by a-root
- 2026-08-13T20:44:30Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/612 (event 01KZYBYDY9J2BSMRNNCM74A3QH)
