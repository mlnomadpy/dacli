---
id: f-all-6-dashboard-sections-rebuilt-on-shadcn-vue-tailwind-with-73-73-tests-green
kind: note
note_kind: finding
created: 2026-07-26T17:40:25Z
created_by: a-tja4fdtr3z
about: [[152]]
severity: moderate
---
# All 6 dashboard sections rebuilt on shadcn-vue + Tailwind with 73/73 tests green and zero test edits
Refactored every section leaf off hand-rolled var(--panel/--border/...) CSS onto the shadcn-vue theme tokens (index.css) + primitives: ProjectCard=Card/CardHeader/CardTitle/CardContent (OverviewSection.vue, ProjectCard.vue), StatusCounts+DAG chip+BurnRate chip+agent-state=Badge, board columns=Card (BoardColumn.vue), swarm=Table/TableHeader/TableRow/TableHead/TableBody/TableCell (AgentSwarm.vue, AgentRow.vue), Retry=Button (ErrorPanel.vue, ConnectionStatus.vue). Behavior preserved verbatim: burn alert yell/hot-bar threshold (BurnRate.vue:38-40,55-60), DAG critical-by-adjacency (DependencyGraph.vue:101-106 unchanged), agent freshness dot buckets + honest state badge (AgentRow.vue), live-count, section 4-state machine. Strategy: kept the semantic marker classes tests select on (.dot/.badge/.chip/.count/.bar/.hot/.node/.edge/.cp-chip/button.retry/[role=group]/.ceiling-line/.done-seg/.rem-seg) so all 73 vitest tests pass UNCHANGED (App.test.ts end-to-end mount incl.). tokens.css trimmed to the 4 status hues + pulse keyframe + .mono + body reset so it no longer shadows the shadcn palette (fixes hover:bg-muted/accent). Verified: npm test:unit 73/73, type-check clean, eslint --max-warnings 0 clean, npm build -> single dist/index.html (.gitkeep intact), go build + go test ./internal/features/dashboard/... green (embed works).
