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
- [ ] Required pull-request CI continues to run go test -race -coverprofile=coverage.out ./..., which replays every committed fuzz regression corpus.
- [ ] Randomized bounded fuzz campaigns run through a lower-frequency scheduled or workflow_dispatch quality workflow rather than on every pull request.
- [ ] All six windows, darwin, and linux amd64/arm64 release targets cross-compile in one Linux job with one checkout, toolchain setup, and artifact download.
- [ ] Workflow contract tests fail if routine PR CI restores a fuzz campaign or a cross-compile matrix, and assert all six target pairs remain present.
- [ ] A GitHub issue comment records the observed baseline: a 4m23s Linux test job with about two minutes allocated to fuzzing and six separately rounded 18-34s cross-compile jobs.
## Log
- 2026-08-28T10:28:23Z claimed by a-fixer-7cpqs2
