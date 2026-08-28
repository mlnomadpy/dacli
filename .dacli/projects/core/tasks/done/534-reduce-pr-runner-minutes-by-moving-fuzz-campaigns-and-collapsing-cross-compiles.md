---
id: t-01M13YH646HH1BWH0NHCM3DQP6
kind: task
created: 2026-08-28T10:27:00Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 2, probable: 4, pessimistic: 6}"
github:
  issue: 853
  repo: mlnomadpy/dacli
---
# Reduce PR runner minutes by moving fuzz campaigns and collapsing cross-compiles
## So that
frequent autonomous agent pull requests retain deterministic release evidence without paying repeated setup and randomized-campaign costs
## Acceptance
- [x] Required pull-request CI continues to run go test -race -coverprofile=coverage.out ./..., which replays every committed fuzz regression corpus.
- [x] Randomized bounded fuzz campaigns run through a lower-frequency scheduled or workflow_dispatch quality workflow rather than on every pull request.
- [x] All six windows, darwin, and linux amd64/arm64 release targets cross-compile in one Linux job with one checkout, toolchain setup, and artifact download.
- [x] Workflow contract tests fail if routine PR CI restores a fuzz campaign or a cross-compile matrix, and assert all six target pairs remain present.
- [x] A GitHub issue comment records the observed baseline: a 4m23s Linux test job with about two minutes allocated to fuzzing and six separately rounded 18-34s cross-compile jobs.
## Log
- 2026-08-28T10:28:23Z claimed by a-fixer-7cpqs2
- 2026-08-28T10:37:27Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/854 (event 01M13YXXTSAHG1H43FB944NA4Z)
- 2026-08-28T10:38:35Z accepted by a-root
- 2026-08-28T10:38:35Z verified by `env GOCACHE=/tmp/dacli-gocache-534-accept go test ./.github/workflows -count=1` (exit 0) in branch dacli/534-reduce-pr-runner-minutes-by-moving-fuzz-campaigns-and-collapsing-cross-compiles at a79bd62d — proves that tree builds, not that the work is in trunk
- 2026-08-28T10:38:35Z deliverable: dacli/534-reduce-pr-runner-minutes-by-moving-fuzz-campaigns-and-collapsing-cross-compiles exists but is NOT in main — closed anyway
- 2026-08-28T10:38:35Z completed by a-root
- 2026-08-28T12:44:55Z a-root: Landing policy override: mode=pr base=main (event 01M13Z6C6FHKCHCH6FM7SA43JA)
- 2026-08-28T12:44:55Z a-root: Integrated via PR https://github.com/mlnomadpy/dacli/pull/854 at merge commit 5e185cd8ea810b815b436e8ef521e3f51730aa14 into main (generation 0) (event 01M13Z6K4K330DFTRN6VPH5C0F)
## Verification Evidence
{"command":"env GOCACHE=/tmp/dacli-gocache-534-accept go test ./.github/workflows -count=1","exit_code":0,"duration_ms":2247,"artifact_hash":"sha256:02a511955643a58e4226e96570cd00d7b84661ca5a14973c7d8198907d8eec27","verifier":"a-root","branch":"dacli/534-reduce-pr-runner-minutes-by-moving-fuzz-campaigns-and-collapsing-cross-compiles","commit_sha":"a79bd62dc37dbc1932fa2d6487ec5c0757ff4f1d"}
