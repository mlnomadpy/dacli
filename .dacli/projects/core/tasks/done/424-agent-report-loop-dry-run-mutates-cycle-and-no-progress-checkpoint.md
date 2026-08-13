---
id: t-01KZXVTQCGGZ7R06JH6JXHZ52W
kind: task
created: 2026-08-13T15:28:39Z
created_by: a-root
owner: a-root
priority: must
github:
  issue: 583
  repo: mlnomadpy/dacli
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
---
# [agent-report] loop --dry-run mutates cycle and no-progress checkpoint
## Context
Adopted from GitHub issue #583.

Reproduction in a workspace with project pm: run 'dacli loop --project pm --width 2 --max-cycles 3 --into master --no-pr --yolo --dry-run' twice. Each preview increments .dacli/loop/pm.txt cycle and .dacli/loop/pm-governor.txt zero_streak even though no agents launch and no work lands. After repeated previews it sets status: halt with reason 'no net progress ... thrash guard tripped', so the next real loop is prevented from running. Expected: --dry-run is read-only and does not alter cycle, governor, rollup, or halt state. Observed on dacli binary /Users/tahabsn/go/bin/dacli on 2026-08-13. Workaround: manually reset only the derived checkpoint after real tasks land.

---
_Reported via `dacli report`._
- dacli: dev
- platform: darwin/arm64
- workspace and run transcript withheld (public upstream) — re-run with --disclose to include them

## Acceptance
- [x] Repeated loop dry-runs do not modify cycle, governor, rollup, halt, or stop state
- [x] Dry-run and live execution share the same planning path and planned commands
- [x] A real loop after any number of dry-runs starts from the same checkpoint it had before preview
- [x] Tests compare workspace state before and after repeated dry-runs and fail on any mutation
- [x] A mutation that restores cycle or zero-streak writes makes the regression fail
## Log
- 2026-08-13T15:34:25Z claimed by a-fixer-sfkfzr
- 2026-08-13T15:36:49Z completed by a-root
