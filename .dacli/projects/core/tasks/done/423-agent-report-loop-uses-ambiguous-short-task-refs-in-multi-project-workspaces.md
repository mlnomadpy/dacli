---
id: t-01KZXVTQABSANB648WHJPMMMXZ
kind: task
created: 2026-08-13T15:28:39Z
created_by: a-root
owner: a-root
priority: should
github:
  issue: 585
  repo: mlnomadpy/dacli
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
---
# [agent-report] loop uses ambiguous short task refs in multi-project workspaces
## Context
Adopted from GitHub issue #585.

Reproduction: in a workspace where multiple projects have task sequence 002/005/007, run 'dacli loop --project pm --width 2 --max-cycles 3 --into master --no-pr --yolo'. The loop correctly selects pm tasks by project, but invokes spawn using short refs 002, 005, and review task 007. Each spawn is refused as ambiguous across ai/auth/bashnota/editor/platform/pm, so two cycles produce no work and the thrash guard halts. Expected: loop should pass project-qualified refs or stable task IDs obtained during project-scoped selection. Actual output includes 'ref 002 is ambiguous' despite --project pm. This makes loop unusable in a multi-project workspace with repeated sequence numbers.

---
_Reported via `dacli report`._
- dacli: dev
- platform: darwin/arm64
- workspace and run transcript withheld (public upstream) — re-run with --disclose to include them

## Acceptance
- [x] Loop spawn and review calls use stable task IDs or project-qualified references from project-scoped selection
- [x] A fixture with duplicate sequence numbers in two projects runs the selected project task without ambiguity
- [x] Review anchors use the same unambiguous reference contract as implementation tasks
- [x] Single-project short-reference behavior remains compatible for operators
- [x] A mutation back to short sequence references makes the multi-project regression fail
## Log
- 2026-08-13T15:51:53Z claimed by a-fixer-f6typj
- 2026-08-13T15:56:42Z accepted by a-root
- 2026-08-13T15:56:42Z closed WITHOUT verification — no --verify command was given
- 2026-08-13T15:56:42Z deliverable: dacli/423-agent-report-loop-uses-ambiguous-short-task-refs-in-multi-project-workspaces exists but is NOT in main — closed anyway
- 2026-08-13T15:56:42Z completed by a-root
- 2026-08-13T16:16:39Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/591 (event 01KZXXE85B2DWSMMH5J4R7M1KN)
- 2026-08-13T16:16:39Z status done proposed by a-fixer-f6typj, applied (event 01KZXXGR6NE0CEV6QG894TJHSB)
