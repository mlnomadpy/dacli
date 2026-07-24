---
id: f-134-vue-dashboard-spa-complete-full-component-tree-12-states-a11y-responsive-48
kind: note
note_kind: finding
created: 2026-07-24T01:15:06Z
created_by: a-j846nahs42
about: [[134]]
severity: major
---
# 134: Vue dashboard SPA complete — full component tree, 12 states, a11y+responsive, 48 Vitest tests green
Built the full DESIGN.md §5 tree in internal/features/dashboard/ui/src/components: AppHeader, OverviewSection→ProjectGrid→ProjectCard→{StatusCounts,BurndownBar}, BoardSection→{ProjectSwitcher,TaskBoard→BoardColumn×4,BurndownChart}, AgentSwarmSection→AgentSwarm→AgentRow, plus shared EmptyPanel/ErrorPanel/SkeletonBlock. Data consumed via the Pinia store (src/stores/dashboard.ts) which polls /api/state every 2000ms (POLL_MS) — the live swarm auto-updates so a running loop's agents appear without restart. All 4 per-surface states (loading/empty/error/live) resolve through src/composables/useSectionState.ts per DESIGN §6.3: a cold failure shows a section error+Retry, a retained snapshot stays live+dims <main>. a11y: one h1, labelled <section>s, real <table> with th scope=col, board columns role=group+aria-label, status conveyed by text not color-only, focus rings, prefers-reduced-motion disables pulse+shimmer. Responsive: overview auto-fill minmax(280,1fr), board 4-col→2×2 <640px, swarm table horizontal-scroll with sticky run column. Gate all green from ui/: npm run type-check (vue-tsc strict), lint (eslint --max-warnings 0), test:unit (48 tests / 12 files incl. store + App end-to-end + BurndownBar/BoardColumn/AgentRow/AgentSwarm/BurndownChart/ProjectGrid), format:check, build (single-file inlined dist/index.html, 93KB per DESIGN §2). Also removed 6 stray committed *.config.js/.d.ts tsc artifacts (one tripped eslint) and gitignored them. go build ./... clean; go test ./internal/features/dashboard/ ok. NOTE: go:embed of the built bundle deferred — see decision note (no CI builds the UI; a hand-committed bundle would drift).
