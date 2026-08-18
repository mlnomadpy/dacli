---
id: d-share-a-fail-closed-agent-run-state-snapshot-between-wip-accounting-and-role
kind: note
note_kind: decision
created: 2026-08-18T12:56:20Z
created_by: a-maintainer-ytrsg6
about: "[[t-01M0AETPE835JWHHS5GA5RE4AW]]"
github:
  issue: 696
  repo: mlnomadpy/dacli
---
# Share a fail-closed agent run-state snapshot between WIP accounting and role removal
## Chose
Share a fail-closed agent run-state snapshot between WIP accounting and role removal
## Rejected
Add a second role-removal-only liveness scan
## Because
One durable snapshot preserves the minted/live/finished contract and returns read failures, preventing capability retraction and WIP routing from drifting again.
