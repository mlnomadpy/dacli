---
id: t-01M11HZWFC5EG8DPVSWWF8XV6B
kind: task
created: 2026-08-27T12:09:22Z
created_by: a-root
owner: a-root
github:
  issue: 807
  repo: mlnomadpy/dacli
estimate: "{optimistic: 1, probable: 2, pessimistic: 3}"
---
# [agent-report] task check rejects documented --verify flag
## Context
Adopted from GitHub issue #807.

The installed CLI reports unknown flag(s): --verify for dacli task check 043 --n N --verify <command>, while the dacli critical-path skill documents that exact handoff. Focused verification was run separately and passed; using supported task check --n syntax.

---
_Reported via `dacli report`._
- dacli: dev
- platform: darwin/arm64
- workspace and run transcript withheld (public upstream) — re-run with --disclose to include them

## Acceptance
## Log
