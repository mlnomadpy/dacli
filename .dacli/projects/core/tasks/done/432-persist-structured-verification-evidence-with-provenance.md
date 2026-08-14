---
id: t-01KZYA0JQJXF62F9SAW5WN3KDR
kind: task
created: 2026-08-13T19:36:31Z
created_by: a-codex-loop-auditor-hxqjcg
owner: a-root
priority: should
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
parent: "[[t-01KZXXZPBXT332W00RP94HTR2K]]"
github:
  issue: 606
  repo: mlnomadpy/dacli
---
# Persist structured verification evidence with provenance
## So that
acceptance records can be audited mechanically instead of relying on a rendered log sentence
## Acceptance
- [x] verification evidence persists command, exit code, duration, artifact hash, verifier identity, branch, and commit SHA in a structured record
- [x] accept and task-check refuse evidence whose artifact hash or verifier identity is missing when the criterion requires command verification
- [x] migration tests read existing string-only task logs without fabricating unavailable evidence fields
## Log
- 2026-08-13T21:18:01Z claimed by a-fixer-3hxnxc
- 2026-08-13T21:52:26Z adopted by a-root (owner a-codex-loop-auditor-hxqjcg orphaned)
- 2026-08-13T21:52:26Z accepted by a-root (applied 1 proposal(s))
- 2026-08-13T21:52:26Z verified by `go test ./internal/store ./internal/features/acceptance ./internal/features/planning` (exit 0) in branch main at f066440 — proves that tree builds, not that the work is in trunk
- 2026-08-13T21:52:26Z deliverable: dacli/432-persist-structured-verification-evidence-with-provenance is merged into main
- 2026-08-13T21:52:26Z completed by a-root
- 2026-08-13T23:51:31Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/626 (event 01KZYGMR2HV1TZ6813PNRHD438)
## Verification Evidence
{"command":"go test ./internal/store ./internal/features/acceptance ./internal/features/planning","exit_code":0,"duration_ms":27501,"artifact_hash":"sha256:74906a6ba51f8015d7b48fd4f07fb409d8c8a2ad62761f1c4d5e3b469101889c","verifier":"a-root","branch":"main","commit_sha":"f066440e09812861d8ad4ab89cb135e793a4bf9a"}
