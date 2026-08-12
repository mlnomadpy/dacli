---
id: t-01KZV40RTE80E83DTWP21GZ5MG
kind: task
created: 2026-08-12T13:54:02Z
created_by: a-codex-loop-auditor-8f0nb8
owner: a-root
priority: must
estimate: "{optimistic: 1, probable: 2, pessimistic: 3}"
depends_on: [365]
github:
  issue: 467
  repo: mlnomadpy/dacli
---
# Strengthen runtime probe cache fingerprint against same-metadata binary replacement
## Acceptance
- [x] internal/store/runtimefiles_test.go replaces a probed fixture binary at the same path with different bytes while preserving file size and modification time, and HydrateRuntimeROProbe returns unknown rather than verified
- [x] internal/store/runtimefiles.go binds cached read-only verdicts to executable content or an equivalently collision-resistant install identity, so a byte-different in-place replacement cannot reuse RuntimeROVerified
- [x] go test ./internal/store/... passes with GOCACHE set to a writable temporary directory
## Log
- 2026-08-12T13:56:30Z adopted by a-root (owner a-codex-loop-auditor-8f0nb8 orphaned)
- 2026-08-12T14:06:03Z claimed by a-root (event 01KZV44V5V8N46R9NAKXPH1ZD3)
- 2026-08-12T14:24:00Z returned to open; the root claim was accidental and no implementation began
- 2026-08-12T15:30:28Z completed by a-root
