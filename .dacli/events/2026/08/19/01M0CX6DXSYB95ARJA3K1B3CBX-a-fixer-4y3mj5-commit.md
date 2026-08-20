---
id: 01M0CX6DXSYB95ARJA3K1B3CBX
kind: event
schema_version: 1
event_kind: commit
created: 2026-08-19T11:41:07Z
created_by: a-fixer-4y3mj5
about: "[[t-01M0AKHSFGWWSMDFCWCE9RYCGQ]]"
origin: agent
applied: true
checksum: sha256:a9d8ed54e1ec053cfdf4b58d8dc282a33db9f8576ec3d9b8ea5cf34f49f9bb32
---
ca7d196 t-01M0AKHSFGWWSMDFCWCE9RYCGQ: recover finished loop proposals

Require a known completed run before root can dismiss an unretired loop-anchor proposal; live, unknown, and never-run actors remain fail-closed.

Mutation: restoring the retired-only guard fails TestRootDismissesFinishedUnretiredProposalOnLoopAnchor with refused-unrelated.
role: fixer
