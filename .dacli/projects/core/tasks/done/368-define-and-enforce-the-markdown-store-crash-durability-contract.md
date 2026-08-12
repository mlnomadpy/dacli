---
id: t-01KZV16EPCRG5DXDSVW08TSSH6
kind: task
created: 2026-08-12T13:04:43Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
depends_on: [364]
github:
  issue: 461
  repo: mlnomadpy/dacli
---
# Define and enforce the markdown store crash-durability contract
## So that
Event and task persistence has an explicit, testable guarantee across process and machine crashes
## Acceptance
- [x] The storage contract explicitly distinguishes atomic rename from power-loss durability
- [x] If power-loss durability is promised, write paths sync file data and the containing directory in the required order
- [x] Focused tests or injected filesystem operations verify the promised ordering and error propagation
- [x] All task, event, and runtime-file writers conform to the documented contract
- [x] go test -race ./... passes
## Log
- 2026-08-12T19:21:36Z claimed by a-codex-maintainer-zf35yj
- 2026-08-12T19:41:59Z accepted by a-root
- 2026-08-12T19:41:59Z verified by `PR #525 merged; go test -race ./... passed on merged main, including mdstore/eventlog/store durability ordering and error propagation tests` (exit 0) in branch main at 588ac26 — proves that tree builds, not that the work is in trunk
- 2026-08-12T19:41:59Z deliverable: dacli/368-define-and-enforce-the-markdown-store-crash-durability-contract is merged into main
- 2026-08-12T19:41:59Z completed by a-root
- 2026-08-12T19:54:17Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/525 (event 01KZVQ85YD3JYYAD5E5XPRE7A8)
- 2026-08-12T19:54:17Z a-root: Integrated via PR https://github.com/mlnomadpy/dacli/pull/525 at merge commit 3a450118f6d537dd35f0a4ad7d127db852176768 into main; local cleanup complete (event 01KZVQSPDA8Q0YP2TDH8JWW1JB)
