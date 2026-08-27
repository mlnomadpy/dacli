---
id: 01M11SEEH3KDNZ9KA9ACKV8284
kind: event
schema_version: 1
event_kind: commit
created: 2026-08-27T14:19:39Z
created_by: a-root
about: "[[t-01M11RCG32CHW4VTAJSPZBTK6F]]"
origin: agent
applied: true
checksum: sha256:66bdf0fda578f58281c081d4bdc78f88f9aeea01934e07ff3ee3641d5099366b
---
0214a212 Add explicit single-harness and hybrid routing policy

Mutation: disabling the harness candidate filter made TestSingleHarnessKeepsImplementationReviewAndFallbackOnCodex fail because the Claude role re-entered the Codex-only candidate set.
role: root
