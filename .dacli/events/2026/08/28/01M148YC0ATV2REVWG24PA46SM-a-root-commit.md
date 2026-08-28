---
id: 01M148YC0ATV2REVWG24PA46SM
kind: event
schema_version: 1
event_kind: commit
created: 2026-08-28T13:28:58Z
created_by: a-root
about: "[[t-01M146BA62817V08T9P6D6REKT]]"
origin: agent
applied: true
checksum: sha256:7b8e7361ed0490ecfb7e35327e0ac6e17e6b6683997aebb81b0d7c16556cbc6f
---
d0442b5f 542: reject stale merged PR generations

Classify a merged PR from a prior task generation as historical rather than canonical, distinguish a current merged-but-nonterminal delivery, and render empty check rollups honestly.

Mutation proof: replacing historical-merged-pr with canonical-pr made TestReconcileDoesNotTreatPriorGenerationMergeAsCanonical fail with: prior-generation merge was called canonical.
role: root
