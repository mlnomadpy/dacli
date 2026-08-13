---
id: t-01KZYQ5ECBFAX3KD08T3VQ5KWV
kind: task
created: 2026-08-13T23:26:22Z
created_by: a-root
owner: a-root
github:
  issue: 637
  repo: mlnomadpy/dacli
---
# [agent-report] Add an enforceable PR-only landing policy for linked GitHub projects
## Context
Adopted from GitHub issue #637.

A GitHub-linked project can currently complete tasks with local 'dacli integrate --into dev', and the normal loop does not require PR review unless the operator remembers '--pr'. Please add a project-level landing policy such as landing.mode=pr and landing.base=dev. When enabled, integrate/ship/loop should refuse local-only merges, push the task branch, open or reuse a PR, wait for required checks/review, and record the PR URL before marking the task landed. The policy should survive loop cycles and make any fallback to local integration an explicit error or override, not the default.

---
_Reported via `dacli report`._
- dacli: dev
- platform: darwin/arm64
- workspace and run transcript withheld (public upstream) — re-run with --disclose to include them

## Acceptance
## Log
