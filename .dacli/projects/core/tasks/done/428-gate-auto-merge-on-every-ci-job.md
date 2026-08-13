---
id: t-01KZY9120MD8NJKG2XE4846F64
kind: task
created: 2026-08-13T19:19:18Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 1, probable: 2, pessimistic: 3}"
parent: "[[t-01KZXS3QNDANVK83M26D3S8W7M]]"
github:
  issue: 600
  repo: mlnomadpy/dacli
---
# Gate auto-merge on every CI job
## So that
the required test check cannot turn green before lint, release, clean-checkout, and cross-platform build evidence succeeds
## Acceptance
- [x] .github/workflows/ci.yml makes the required test job depend on test-matrix, lint, clean-checkout, release-snapshot, and cross-compile
- [x] the required test job fails unless every dependency result equals success, including every cross-compile matrix leg
- [x] an automated workflow-contract test proves a failed or cancelled downstream gate cannot produce a successful required test check
- [x] GitHub main branch protection continues requiring the stable test context and the fix PR passes every CI job before merge
## Log
- 2026-08-13T19:20:03Z claimed by a-codex-maintainer-xytv4d
- 2026-08-13T19:31:54Z accepted by a-root
- 2026-08-13T19:31:54Z verified by `GOCACHE=/private/tmp/dacli-428-main-gocache go test ./.github/workflows` (exit 0) in branch main at 937138b — proves that tree builds, not that the work is in trunk
- 2026-08-13T19:31:54Z deliverable: dacli/428-gate-auto-merge-on-every-ci-job is merged into main
- 2026-08-13T19:31:54Z completed by a-root
- 2026-08-13T19:38:29Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/601 (event 01KZY9CXVECNH6HRAB5FY1XY0G)
