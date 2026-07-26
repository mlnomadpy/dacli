---
id: d-084-filed-task-150-dag-view-falsely-highlights-non-critical-edges-as-the-single
kind: note
note_kind: decision
created: 2026-07-26T15:47:33Z
created_by: a-avy9rqtfdw
about: [[084]]
---
# 084: filed task 150 (DAG view falsely highlights non-critical edges) as the single highest-value evidence-based change
## Chose
084: filed task 150 (DAG view falsely highlights non-critical edges) as the single highest-value evidence-based change
## Rejected
Filing the /api/graph cross-project seq-collision in byRef (graph.go:114 %03d key overwrites across projects), or filing that DependencyGraph.vue has no unit test, or re-filing the burn population mismatch (already task 149)
## Because
The byRef %03d collision is a faithful mirror of the pre-existing cmdCriticalPath pattern (insight.go:471) and is only reachable via the standalone /api/graph with an empty ?project= -- the SPA never calls that; it reads the correctly per-project-scoped graph embedded in /api/state (dashboard.go:418 buildGraph(w, p.Slug)), so real impact is nil. 'No unit test' is a symptom, not a defect, and is folded into task 150's acceptance instead of standing alone. The burn mismatch is already task 149. Task 150 is a genuine CORRECTNESS defect in shipped, merged code (145): DependencyGraph.vue:111 colors an edge critical from (from.critical && to.critical) rather than adjacency on the ordered graph.critical_path (available at types.ts:72 but used only for a count), so a positive-slack redundant edge between two on-path nodes is painted red -- actively misleading the very spawn decision the view exists to guide (caption DependencyGraph.vue:218). It is evidence-grounded (file:line + a concrete A->C->B repro), squarely in the frontend-reviewer scope, and fully unit-testable.
