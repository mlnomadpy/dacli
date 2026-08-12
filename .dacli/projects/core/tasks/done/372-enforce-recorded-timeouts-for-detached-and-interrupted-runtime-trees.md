---
id: t-01KZV2X41PEMSRRGZYY1Y8PQ92
kind: task
created: 2026-08-12T13:34:34Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 3, probable: 8, pessimistic: 12}"
depends_on: [369]
github:
  issue: 465
  repo: mlnomadpy/dacli
---
# Enforce recorded timeouts for detached and interrupted runtime trees
## So that
A silent coding CLI cannot outlive a finalized run, retain claims, or hang the autonomous loop indefinitely
## Acceptance
- [x] A detached spawn that exceeds timeout_s terminates its exact process tree and records a timed-out outcome even after the launching dacli process exits
- [x] Interrupting a foreground spawn cannot orphan the coding CLI leader or helper descendants
- [x] agents, wait, kill, claim-overlap, and outcome.md derive one consistent live/finalized state; a finalized run with a live recorded leader is never hidden as no live agents
- [x] A regression fixture reproduces a silent leader with helper descendants and proves its claim becomes reusable after timeout
- [x] PID start identity prevents any cleanup path from signaling a recycled unrelated PID or PGID
- [x] go test -race ./... passes
## Log
- 2026-08-12T13:57:58Z claimed by a-codex-maintainer-nhx5wh
- 2026-08-12T14:10:06Z completed by a-root
