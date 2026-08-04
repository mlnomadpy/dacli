---
id: f-dag-view-highlights-non-critical-edges-uses-from-critical-to-critical-not
kind: note
note_kind: finding
created: 2026-08-04T00:37:40Z
created_by: a-avy9rqtfdw
about: "[[t-01KY60QM1Y7DK05WXB954YNDHJ]]"
source_event: 01KYFHQTR8RYQHCSQDG8BNNB2J
---
# DAG view highlights non-critical edges: uses from.critical && to.critical, not adjacency on critical_path
internal/features/dashboard/ui/src/components/DependencyGraph.vue:111 computes an edge's 'critical' flag as (from.node.critical && to.node.critical). That marks ANY edge between two on-path nodes as critical/red, even when that specific edge carries positive slack. The server exposes the true zero-slack chain as an ORDERED list graph.critical_path (types.ts:72, graph.go:196), but the component uses it only for a count (DependencyGraph.vue:128), never for edge coloring. Repro: tasks A(Te5),C(Te1),B(Te5) with deps A->C, C->B, and a redundant direct A->B. CPM path is A->C->B (duration 11); the A->B edge has slack 1 (non-critical). All three nodes are critical, so DependencyGraph paints the A->B edge red anyway. This misleads the operator that the view exists to serve -- caption at DependencyGraph.vue:218 says the highlighted chain is where to 'spawn children here first'. Correct fix: mark an edge critical iff (from,to) is a consecutive pair in graph.critical_path. Component has no unit test (no DependencyGraph.test.ts) so this is uncaught. Shipped in task 145 (commit a70e26f).
