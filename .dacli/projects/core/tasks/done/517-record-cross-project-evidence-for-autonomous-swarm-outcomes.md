---
id: t-01M11HZVZ5P60GBNEHGC5CQX00
kind: task
created: 2026-08-27T12:09:21Z
created_by: a-root
owner: a-root
github:
  issue: 818
  repo: mlnomadpy/dacli
depends_on: "[t-01M11HZW2DMC87Z8TNWGX7DFQ1]"
estimate: "{optimistic: 1, probable: 2, pessimistic: 3}"
---
# Record cross-project evidence for autonomous swarm outcomes
## Context
Adopted from GitHub issue #818.

## Problem

Repository research currently exposes self-hosting evidence and a composite human-adopter persona, so reviewers can reasonably but incorrectly infer that dacli has not been exercised on other products. The project needs an evidence model suited to its actual agent-facing audience.

## Design direction

Document anonymized, reproducible cross-project outcome evidence without disclosing private repositories or credentials. Measure whether agent-run swarms deliver accepted product changes, not whether humans memorize the CLI. Separate technical validation, cross-project workflow validation, independent operator adoption, and commercial evidence.

## Acceptance
- [x] A documented case-study template records repository archetype/stack, goal, runtimes/models, task and PR outcomes, elapsed time, token/cost evidence, interventions, refusals/recoveries, CI/review results, and release outcome.
- [x] At least one anonymized cross-project run is represented, or the documentation clearly marks the evidence slot pending rather than inventing data.
- [x] docs/research distinguishes agent-facing product validation from human CLI onboarding research.
- [x] README or the docs index links to the evidence without making unsupported quantitative claims.
- [x] Private repository names, prompts, credentials, and proprietary source are excluded.
- [x] Documentation checks pass.
## Log
- 2026-08-27T12:09:33Z dependency edit by a-root (event 01M11J0743SG7JY574HNN87ERA)
- 2026-08-27T12:36:39Z accepted by a-root
- 2026-08-27T12:36:39Z verified by `go test ./docs` (exit 0) in branch main at f74df2a — proves that tree builds, not that the work is in trunk
- 2026-08-27T12:36:39Z deliverable: dacli/517-record-cross-project-evidence-for-autonomous-swarm-outcomes is merged into main
- 2026-08-27T12:36:39Z completed by a-root
- 2026-08-27T12:45:18Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/824 (event 01M11K4TQCY1MBMVBR710BN0NN)
## Verification Evidence
{"command":"go test ./docs","exit_code":0,"duration_ms":62,"artifact_hash":"sha256:f4796ba07855189b7bb28c2f14a6290f878f2a1a7bdb4e3a6a5a93f908459903","verifier":"a-root","branch":"main","commit_sha":"f74df2ab7e5b6c7d7c396d5b3540ce67c35d13d0"}
