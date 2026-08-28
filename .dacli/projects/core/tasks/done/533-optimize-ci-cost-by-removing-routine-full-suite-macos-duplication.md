---
id: t-01M13X19WKEC3MXWMS475GCSR2
kind: task
created: 2026-08-28T10:00:51Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 2, probable: 4, pessimistic: 7}"
github:
  issue: 851
  repo: mlnomadpy/dacli
---
# Optimize CI cost by removing routine full-suite macOS duplication
## So that
autonomous dacli loops preserve meaningful cross-platform evidence without exhausting hosted-runner budgets
## Acceptance
- [x] The pull-request CI required check runs the full Go/frontend verification on one Linux runner rather than a Linux/macOS full-suite matrix.
- [x] Merges do not immediately duplicate the already-tested pull-request pipeline on main; manual recovery remains available through workflow_dispatch.
- [x] Native macOS validation is retained as a narrow platform-sensitive or release gate, while darwin amd64 and arm64 cross-compilation remains on Linux.
- [x] Workflow contract tests fail if routine PR CI reintroduces macos-latest or push-to-main duplication.
- [x] The issue documents expected cost reduction, the public-repository billing caveat, and the residual risk of moving native macOS coverage out of every PR.
## Log
- 2026-08-28T10:02:33Z claimed by a-fixer-fv8pny
- 2026-08-28T10:08:28Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/852 (event 01M13XBMWJVRN5Z985DEXWSBQ8)
- 2026-08-28T10:18:11Z accepted by a-root
- 2026-08-28T10:18:11Z verified by `env GOCACHE=/tmp/dacli-gocache-533-accept go test ./.github/workflows -count=1` (exit 0) in branch dacli/533-optimize-ci-cost-by-removing-routine-full-suite-macos-duplication at ee268a76 — proves that tree builds, not that the work is in trunk
- 2026-08-28T10:18:11Z deliverable: dacli/533-optimize-ci-cost-by-removing-routine-full-suite-macos-duplication exists but is NOT in main — closed anyway
- 2026-08-28T10:18:11Z completed by a-root
- 2026-08-28T10:28:06Z a-root: Landing policy override: mode=pr base=main (event 01M13Y10ZXQXQDCKCQ9CKQY7BX)
- 2026-08-28T10:28:06Z a-root: Integrated via PR https://github.com/mlnomadpy/dacli/pull/852 at merge commit 2a3f51ef7bca01c484b57c108df4d4fe7f89af83 into main (generation 0) (event 01M13Y18BZYMW4NHTZPHH6MQQW)
## Verification Evidence
{"command":"env GOCACHE=/tmp/dacli-gocache-533-accept go test ./.github/workflows -count=1","exit_code":0,"duration_ms":3338,"artifact_hash":"sha256:c92d3600d6e48b58f8e710937740ac69373a65384fb82408cffcc76270683bf7","verifier":"a-root","branch":"dacli/533-optimize-ci-cost-by-removing-routine-full-suite-macos-duplication","commit_sha":"ee268a7654a953fd62fc38ecca4a5801d27f1d33"}
