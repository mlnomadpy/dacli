---
id: t-01M0ZCAPT8TKDV88VZ7R1B1WWR
kind: task
created: 2026-08-26T15:51:56Z
created_by: a-root
owner: a-root
github:
  issue: 793
  repo: mlnomadpy/dacli
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
---
# [agent-report] scheduler rejects legacy hard dependency types after loading the workspace
## Context
Adopted from GitHub issue #793.

Observed on a mature workspace whose persisted tasks contain dependency entries such as [034:hard,...]. Both dacli critical-path --project <project> and dacli next --project <project> --parallel N fail with 'dependency graph: unknown dependency type HARD' instead of scheduling. The task files were created by dacli and task show still renders them as hard dependencies. Manual workaround: explicitly inspect task dependencies and launch only independently verified tasks; there is no safe scheduling output. Expected: normalize supported legacy hard values to the current dependency enum during load/migration, or doctor should identify and offer an audited migration before schedulers run. Acceptance criteria: regression-load a task containing :hard, prove next and critical-path succeed with the intended finish-to-start edge, and prove truly unknown values still fail closed. Non-goal: silently dropping dependency edges.

---
_Reported via `dacli report`._
- dacli: dev
- platform: darwin/arm64
- workspace and run transcript withheld (public upstream) — re-run with --disclose to include them

## Acceptance
- [ ] Loading a persisted `:hard` dependency normalizes it to the supported finish-to-start dependency type without dropping the edge.
- [ ] `dacli next --parallel N` and `dacli critical-path` schedule a fixture containing the legacy value successfully.
- [ ] A truly unknown dependency value still fails closed with an actionable diagnostic.
- [ ] Doctor or migration behavior is documented if persisted files are rewritten rather than normalized at read time.
- [ ] Mutation evidence and the full repository verification gates pass.
## Log
