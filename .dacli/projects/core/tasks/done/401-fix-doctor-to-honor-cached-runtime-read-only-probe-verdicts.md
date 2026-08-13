---
id: t-01KZVVDTDR85KRDRY0574ZW1RW
kind: task
created: 2026-08-12T20:43:07Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 1, probable: 2, pessimistic: 3}"
github:
  issue: 543
  repo: mlnomadpy/dacli
---
# Fix doctor to honor cached runtime read-only probe verdicts
## Acceptance
- [x] After runtime doctor verifies the cc read-only sandbox, dacli doctor does not report grant-runtime-mismatch for roles using cc
- [x] Doctor hydrates a cached probe only when the runtime declaration and installed binary fingerprint still match
- [x] A failed or missing probe, including the current codex-ro failure, still reports the read-only enforcement mismatch
- [x] Focused regression coverage reproduces the current runtime-doctor verified followed by doctor mismatch sequence
## Log
- 2026-08-12T20:47:34Z claimed by a-codex-maintainer-c1rsc9
- 2026-08-13T09:05:36Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/544 (event 01KZVVX0GQM9H1BGSQZWYMCHXF)
- 2026-08-13T09:05:51Z accepted by a-root
- 2026-08-13T09:05:51Z verified by `GOCACHE=/private/tmp/dacli-go-cache go test ./internal/features/insight -run TestDoctorUsesRuntimeDoctorReadOnlyVerdict -count=1` (exit 0) in branch main at bc86249 — proves that tree builds, not that the work is in trunk
- 2026-08-13T09:05:51Z deliverable: dacli/401-fix-doctor-to-honor-cached-runtime-read-only-probe-verdicts exists but is NOT in main — closed anyway
- 2026-08-13T09:05:51Z completed by a-root
