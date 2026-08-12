---
id: t-01KZV5RSPJ5PSG74VHH2Q4A7JQ
kind: task
created: 2026-08-12T14:24:38Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
github:
  issue: 471
  repo: mlnomadpy/dacli
---
# Reset the loop thrash streak when trunk advances between invocations
## Acceptance
- [x] starting loop with a persisted halted zero_streak and a current trunk marker newer than the persisted marker clears the no-progress halt and records the new marker
- [x] starting loop with the same trunk marker preserves the halt, so retrying an unchanged stalled cycle remains refused
- [x] loop status explains whether recovery came from observed trunk advancement or an explicit operator reset, and orchestration governor/state tests pass
## Log
- 2026-08-12T15:26:20Z claimed by a-codex-maintainer-2amnk2
- 2026-08-12T15:34:48Z completed by a-root
