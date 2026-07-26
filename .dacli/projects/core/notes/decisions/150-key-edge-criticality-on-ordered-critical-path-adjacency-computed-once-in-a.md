---
id: d-150-key-edge-criticality-on-ordered-critical-path-adjacency-computed-once-in-a
kind: note
note_kind: decision
created: 2026-07-26T17:00:38Z
created_by: a-17bm85cpf7
about: [[150]]
---
# 150: key edge criticality on ordered critical_path adjacency, computed once in a Set
## Chose
150: key edge criticality on ordered critical_path adjacency, computed once in a Set
## Rejected
keeping from.node.critical && to.node.critical, or recomputing the pair string inside the per-edge flatMap
## Because
critical_path is an ordered id list (types.ts:71); the CPM chain, not the set of critical nodes, defines which EDGES carry zero slack. A redundant A->B when the path is A->C->B joins two critical nodes yet has positive slack, so both-endpoints logic falsely paints it red. Consecutive (cp[i],cp[i+1]) pairs are exactly the critical adjacencies; a single computed Set keyed 'from->to' is O(1) per edge and mirrors the edgePaths key format.
