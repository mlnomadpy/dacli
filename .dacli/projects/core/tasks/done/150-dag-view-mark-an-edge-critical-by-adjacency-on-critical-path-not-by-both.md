---
id: t-01KYFHR19HAA9FVN6BF6E0A845
kind: task
created: 2026-07-26T15:47:21Z
created_by: a-avy9rqtfdw
owner: a-root
priority: must
github:
  issue: 246
  repo: mlnomadpy/dacli
---
# DAG view: mark an edge critical by adjacency on critical_path, not by both endpoints being critical
## Acceptance
- [x] DependencyGraph.vue edge 'critical' flag is true iff (from,to) is a consecutive pair in graph.critical_path (built from an ordered set of the path), not from.node.critical && to.node.critical
- [x] A redundant edge between two on-path but non-adjacent nodes (e.g. direct A->B when the chain is A->C->B) renders as a normal, non-critical edge
- [x] Add a DependencyGraph unit test (currently none) covering: adjacent-on-path edge is critical; non-adjacent both-critical edge is NOT; SS edge dasharray preserved
- [x] npm run test:unit green in internal/features/dashboard/ui
## Log
- 2026-07-26T16:58:33Z claimed by a-17bm85cpf7
- 2026-07-26T17:01:13Z adopted by a-root (owner a-avy9rqtfdw orphaned)
- 2026-07-26T17:01:13Z accepted by a-root
- 2026-07-26T17:01:13Z completed by a-root
- 2026-08-03T22:38:15Z a-17bm85cpf7: PR opened: https://github.com/mlnomadpy/dacli/pull/109 (event 01KYFNYXHV5A0TEEFDGTAJKA4F)
