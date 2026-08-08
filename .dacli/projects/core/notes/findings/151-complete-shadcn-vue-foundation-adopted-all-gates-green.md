---
id: f-151-complete-shadcn-vue-foundation-adopted-all-gates-green
kind: note
note_kind: finding
created: 2026-07-26T17:16:52Z
created_by: a-12vbv8bcwh
about: [[151]]
severity: moderate
---
# 151 complete: shadcn-vue foundation adopted, all gates green
Branch dacli/151-... commit 1bf1941 by a-12vbv8bcwh. AC1: shadcn-vue foundation authored by hand (init CLI is interactive/network-fetch, unusable headless) — components.json (new-york, cssVariables), @tailwindcss/vite added to vite.config.ts, src/assets/index.css imports tailwindcss+tw-animate-css and defines the DARK mission-control theme on :root mapping shadcn tokens 1:1 to the palette (#0f1115 background/--bg, #161a22 card/--panel, #262b36 border, #e6e9ef foreground, #8b93a7 muted-foreground, #4f8cff primary+ring/--active blue accent, #e5534b destructive, #3fb950 success) with @theme inline exposing them as Tailwind utilities; src/lib/utils.ts cn(); base components under src/components/ui/{button,card,badge,table,progress,separator,tooltip} (Reka UI primitives + cva). Single-file build preserved: viteSingleFile still emits one dist/index.html (129 kB, was 107) with JS+CSS inlined; grep confirms NO external <script>/<link> or url()/@import CDN — only SVG/MathML namespace URIs. AC2: npm run build (type-check+vite) green, npm run test:unit 73 passed (added src/components/ui/__tests__/ui.test.ts, 11 smoke tests mounting every base component), eslint --max-warnings 0 clean (added an override turning off vue/multi-word-component-names for src/components/ui/** per shadcn-vue docs), prettier --check clean, go build ./... OK. Existing sections unchanged: tokens.css kept and loaded after index.css (unlayered, wins over Tailwind base); App.test.ts + all prior tests still pass. Sections not yet refactored onto the components — that is out of scope for this foundation task.
