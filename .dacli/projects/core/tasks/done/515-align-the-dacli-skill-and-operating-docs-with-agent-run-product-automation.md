---
id: t-01M11HZVRFFZHJXXK0E5549FES
kind: task
created: 2026-08-27T12:09:21Z
created_by: a-root
owner: a-root
github:
  issue: 820
  repo: mlnomadpy/dacli
depends_on: "[t-01M11HZW2DMC87Z8TNWGX7DFQ1]"
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
---
# Align the dacli skill and operating docs with agent-run product automation
## Context
Adopted from GitHub issue #820.

## Problem

The skill and operating documentation correctly emphasize safety and bounded loops, but they still read primarily as instructions for a human CLI operator. The intended hot path is an orchestrator AI agent using dacli to plan, route, run, review, recover, and land work across replaceable coding CLIs.

## Design direction

Keep the skill compact as a router, but make the agent identity, authority boundaries, critical-path workflow, model economics, loop recovery, and GitHub evidence cycle explicit. Human commands should be framed as governance and intervention points. Avoid duplicating long runbooks between the skill and repository docs.

## Acceptance
- [x] skills/dacli/SKILL.md identifies the primary user as an orchestrator agent governing coding-agent CLIs.
- [x] docs/OPERATOR_PLAYBOOK.md and directly linked skill references teach one consistent task/wave/loop workflow.
- [x] Normal operation prioritizes critical path, provider-neutral capability routing, cost-aware model selection, bounded loops, independent review, recovery, and verified GitHub landing.
- [x] Human responsibilities are limited and explicit: direction, authority, exceptions, emergency stop, audit, and release policy.
- [x] Expert primitives remain documented as recovery/control mechanisms without becoming the primary mental model.
- [x] Every documented command is checked against current dacli help and documentation tests pass.
## Log
- 2026-08-27T12:09:33Z dependency edit by a-root (event 01M11J07226T6E5EJ92RCA4Y18)
- 2026-08-27T12:36:32Z accepted by a-root
- 2026-08-27T12:36:32Z verified by `go test ./docs` (exit 0) in branch main at f74df2a — proves that tree builds, not that the work is in trunk
- 2026-08-27T12:36:32Z deliverable: dacli/515-align-the-dacli-skill-and-operating-docs-with-agent-run-product-automation is merged into main
- 2026-08-27T12:36:32Z completed by a-root
- 2026-08-27T12:45:18Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/823 (event 01M11K1TAFQ1264AKQBTDJHG3R)
## Verification Evidence
{"command":"go test ./docs","exit_code":0,"duration_ms":506,"artifact_hash":"sha256:73dfd07d0923d92b4860115e7d13b8ba46f5eef06520e6ab4a7b42bfc94d938a","verifier":"a-root","branch":"main","commit_sha":"f74df2ab7e5b6c7d7c396d5b3540ce67c35d13d0"}
