---
id: f-150-complete-dag-edge-criticality-now-keyed-on-critical-path-adjacency-all-4
kind: note
note_kind: finding
created: 2026-07-26T17:00:51Z
created_by: a-17bm85cpf7
about: [[150]]
severity: moderate
---
# 150 complete: DAG edge criticality now keyed on critical_path adjacency; all 4 acceptance met
Commit abdcc4a on branch dacli/150. DependencyGraph.vue: new criticalEdges computed builds a Set of consecutive 'from->to' pairs from the ordered graph.critical_path; edgePaths marks critical = criticalEdges.has(key) instead of from.node.critical && to.node.critical (was DependencyGraph.vue:111). AC1 met (adjacency, not both-endpoints). AC2 met: new test 'marks an edge critical by adjacency...' proves a redundant A->B when the path is A->C->B (all 3 nodes critical) renders 3 edges, only 2 critical. AC3 met: added 2 DependencyGraph unit tests (adjacency-critical + non-adjacent-both-critical NOT critical; plus SS-dasharray-preserved-on-critical-edge test confirms '4 3' survives with the critical class). AC4 met: npm run test:unit green in internal/features/dashboard/ui — 14 files, 62 tests pass (DependencyGraph 8, was 6); vue-tsc type-check clean. No Go files touched.
