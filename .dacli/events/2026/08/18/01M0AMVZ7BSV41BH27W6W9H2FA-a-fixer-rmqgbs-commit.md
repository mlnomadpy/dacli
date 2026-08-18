---
id: 01M0AMVZ7BSV41BH27W6W9H2FA
kind: event
schema_version: 1
event_kind: commit
created: 2026-08-18T14:37:07Z
created_by: a-fixer-rmqgbs
about: "[[t-01M0AKHSFGWWSMDFCWCE9RYCGQ]]"
origin: agent
applied: true
checksum: sha256:0b6b0de5d5e0cd903fe8cb7b9b09ad26a0abc0d22a1bfb4ba87c5e630b02c5ae
---
0071089 t-01M0AKHSFGWWSMDFCWCE9RYCGQ: recover retired loop-anchor proposals

The loop anchor has synthetic owner loop, so the prior retired-owner recovery could never resolve its stale reviewer proposals. Permit root to append an audited dismissal only for a recognized loop anchor and a retired actor; ordinary, unresolved, corrupt, and read-only cases stay refused.

Mutation evidence: before the guard, GOCACHE=/tmp/dacli-go-build-cache go test ./internal/features/collab -run '^TestRootDismissesRetiredProposalOnLoopAnchor$' -count=1 failed: root could not dismiss retired loop proposal: a-root cannot dismiss event ...: refused-unrelated.
role: fixer
