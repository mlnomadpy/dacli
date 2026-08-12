---
id: 01KZVA2CXYZENEM9609HKCPPRZ
kind: event
event_kind: commit
created: 2026-08-12T15:39:47Z
created_by: a-codex-maintainer-vxzmpg
about: "[[t-01KZV5RSNF2C50DHC8HGSPJMNX]]"
origin: agent
applied: true
---
e698348 378: derive loop worker timeouts from estimates

Add --worker-timeout as an explicit loop policy override and otherwise pass a five-minutes-per-Te timeout to implementation and review spawns, retaining a five-minute floor. Document the policy beside the driver and assert complete spawn argument vectors.

Red test evidence: mutating the derived timeout back to 300 failed TestLoopDerivesWorkerTimeoutFromEachTaskEstimate: got [spawn --task 001 --role fixer --detach --worktree --pr --timeout 300], want the same vector with --timeout 1800.
role: codex-maintainer
