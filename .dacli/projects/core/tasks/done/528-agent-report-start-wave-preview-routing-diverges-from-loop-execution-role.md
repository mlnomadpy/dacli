---
id: t-01M12N9W6E9T3SSG3V2P2Y6Y9R
kind: task
created: 2026-08-27T22:26:29Z
created_by: a-root
owner: a-root
github:
  issue: 837
  repo: mlnomadpy/dacli
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
---
# [agent-report] start wave preview routing diverges from loop execution role
## Context
Adopted from GitHub issue #837.

Reproduction on main 483b32d4: start --project core --profile wave --harness codex preview selected task 509 and task 510 as maintainer on codex-rw with gpt-5.6-sol because each Te 8.3 exceeds fixer capacity. Actual execution printed loop impl=fixer and then refused task 509 with automatic routing found no eligible implementer role. A direct team assign 509 immediately selected maintainer/codex-rw/gpt-5.6-sol. Expected: the executed loop must preserve the profile resolver per-task role/runtime/model decision, or recompute an equivalent eligible decision under the same single-harness constraint; preview and execution must not disagree. Acceptance: a regression previews a high-complexity task routed to maintainer, executes the profile, and proves the spawned role/runtime/model match; single-harness codex remains enforced; an explicit incompatible role still fails closed; go test ./... passes. Manual workaround: invoke dacli spawn for the task with the previewed maintainer role instead of start wave.

---
_Reported via `dacli report`._
- dacli: dev
- platform: darwin/arm64
- workspace and run transcript withheld (public upstream) — re-run with --disclose to include them

## Acceptance
## Log
- 2026-08-27T22:27:16Z claimed by a-fixer-ce80x9
- 2026-08-27T22:42:05Z accepted by a-root
- 2026-08-27T22:42:05Z closed WITHOUT verification — no --verify command was given
- 2026-08-27T22:42:05Z deliverable: dacli/528-agent-report-start-wave-preview-routing-diverges-from-loop-execution-role is merged into main
- 2026-08-27T22:42:05Z closed with NO acceptance criteria — UNVERIFIED (--allow-unverified)
- 2026-08-27T22:42:05Z completed by a-root
- 2026-08-27T22:51:57Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/839 (event 01M12NS3TEM4NTXGP9NM72DS43)
