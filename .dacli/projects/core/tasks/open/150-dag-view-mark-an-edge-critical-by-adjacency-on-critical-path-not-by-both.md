---
id: t-01KYFHR19HAA9FVN6BF6E0A845
kind: task
created: 2026-07-26T15:47:21Z
created_by: a-avy9rqtfdw
owner: a-avy9rqtfdw
priority: must
---
# DAG view: mark an edge critical by adjacency on critical_path, not by both endpoints being critical
## Acceptance
- [ ] DependencyGraph.vue edge 'critical' flag is true iff (from,to) is a consecutive pair in graph.critical_path (built from an ordered set of the path), not from.node.critical && to.node.critical
- [ ] A redundant edge between two on-path but non-adjacent nodes (e.g. direct A->B when the chain is A->C->B) renders as a normal, non-critical edge
- [ ] Add a DependencyGraph unit test (currently none) covering: adjacent-on-path edge is critical; non-adjacent both-critical edge is NOT; SS edge dasharray preserved
- [ ] npm run test:unit green in internal/features/dashboard/ui
## Log
