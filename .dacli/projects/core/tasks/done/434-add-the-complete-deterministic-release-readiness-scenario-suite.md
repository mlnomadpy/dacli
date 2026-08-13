---
id: t-01KZYA0JWPDRV1WVXTE08F1W0X
kind: task
created: 2026-08-13T19:36:31Z
created_by: a-codex-loop-auditor-hxqjcg
owner: a-root
priority: should
estimate: "{optimistic: 5, probable: 8, pessimistic: 13}"
parent: "[[t-01KZXXZPBXT332W00RP94HTR2K]]"
github:
  issue: 608
  repo: mlnomadpy/dacli
---
# Add the complete deterministic release-readiness scenario suite
## So that
the single happy-path self-host fixture also covers the failure scenarios named by issue 437
## Acceptance
- [x] internal/scenarios contains deterministic offline fixtures for feature work, regression repair, dependency failure, conflicting edits, and malicious instructions
- [x] one documented command runs every scenario and reports outcome assertions rather than command-call counts
- [x] CI runs the suite and each scenario has a demonstrated mutation that makes its assertion fail
## Log
- 2026-08-13T19:56:09Z claimed by a-codex-maintainer-7ebfye
- 2026-08-13T20:30:49Z adopted by a-root (owner a-codex-loop-auditor-hxqjcg orphaned)
- 2026-08-13T20:30:49Z accepted by a-root (applied 1 proposal(s))
- 2026-08-13T20:30:49Z verified by `go test ./internal/scenarios` (exit 0) in branch main at 7874016 — proves that tree builds, not that the work is in trunk
- 2026-08-13T20:30:49Z deliverable: dacli/434-add-the-complete-deterministic-release-readiness-scenario-suite is merged into main
- 2026-08-13T20:30:49Z completed by a-root
- 2026-08-13T20:44:30Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/613 (event 01KZYBYQG5CX3Z8FDB6P36907K)
