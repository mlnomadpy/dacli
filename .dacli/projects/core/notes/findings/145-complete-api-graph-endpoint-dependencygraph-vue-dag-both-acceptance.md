---
id: f-145-complete-api-graph-endpoint-dependencygraph-vue-dag-both-acceptance
kind: note
note_kind: finding
created: 2026-07-24T11:29:46Z
created_by: a-gbyc86v99b
about: [[145]]
severity: moderate
---
# 145 complete: /api/graph endpoint + DependencyGraph.vue DAG, both acceptance criteria met and tested
Go: internal/features/dashboard/graph.go adds buildGraph(w,project) — every task becomes a node (by status) and every resolvable depends_on an edge (the DAG, always drawn), then spm.ComputeCPM overlays the critical path over the open+estimated subset (same subset dacli next/critical-path use). Degrades with graphView.Note (unestimated open task / cycle) instead of refusing. Exposed at GET /api/graph?project= (envelope, mirrors /api/burn) AND embedded per-project in projectView.Graph so the SPA's single /api/state poll carries it. Handler tests in graph_test.go (TestAPIGraph chain A→B→C all-critical + done node, TestAPIStateEmbedsGraph, TestAPIGraphDegradesWhenUnestimated, TestGraphEmptyWorkspaceIsZeroSafe) all green. Vue: DependencyGraph.vue draws an SVG layered DAG (columns=dependency depth via longest-path, cycle-guarded), nodes colored by status, critical path highlighted on nodes AND edges, SS edges dashed, legend + degrade note + empty state; DagSection.vue mirrors BoardSection (project switcher, shared selectedSlug) and is wired into App.vue after the board. 6 vitest specs in DependencyGraph.test.ts green. Full internal Go suite + 58 UI tests + type-check + eslint + prettier all clean. NOTE: real-binary smoke of 'dacli dashboard' was blocked by the headless sandbox (arbitrary-path exec needs approval); the HTTP surface is instead driven end-to-end via httptest against the real newHandler and the component via a real vitest mount.
