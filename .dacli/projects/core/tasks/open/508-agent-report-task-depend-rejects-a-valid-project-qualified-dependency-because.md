---
id: t-01M1068M8HJ9G8XCXMEMVE2V8D
kind: task
created: 2026-08-26T23:25:11Z
created_by: a-root
owner: a-root
github:
  issue: 800
  repo: mlnomadpy/dacli
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
---
# [agent-report] task depend rejects a valid project-qualified dependency because unrelated legacy tasks contain ambiguous unqualified refs. Repro: adding an FS dependency between two project-qualified tasks aborts while validating unrelated stored dependencies such as 002 or 014 across other projects. Expected: validate the changed task/edge or resolve stored refs within their owning project; unrelated legacy ambiguity must not prevent all future graph edits.
## Context
Adopted from GitHub issue #800.

---
_Reported via `dacli report`._
- dacli: dev
- platform: darwin/arm64
- workspace and run transcript withheld (public upstream) — re-run with --disclose to include them

## Acceptance
- [ ] A multi-project fixture contains ambiguous legacy numeric dependency refs unrelated to the task being changed.
- [ ] `task depend <project-qualified-task> --on <project-qualified-task>` validates and persists the requested edge without resolving unrelated legacy refs globally.
- [ ] Cycle detection and missing-target validation still reject invalid edges within the changed task's reachable dependency graph.
- [ ] Existing legacy refs resolve in their owning project where possible and produce a scoped diagnostic when that specific edge is inspected.
- [ ] Mutation evidence and the full repository verification gates pass.
## Log
