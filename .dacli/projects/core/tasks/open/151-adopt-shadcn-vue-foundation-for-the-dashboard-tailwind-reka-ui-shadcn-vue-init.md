---
id: t-01KYFN5MADAEF2FDTJSV9TPQPJ
kind: task
created: 2026-07-26T16:47:12Z
created_by: a-root
owner: a-root
priority: must
depends_on: [149, 150]
---
# Adopt shadcn-vue foundation for the dashboard: Tailwind + Reka UI + shadcn-vue init, dark mission-control theme
## So that
the dashboard gets a polished, accessible, consistent component system it owns, instead of hand-rolled CSS
## Acceptance
- [ ] shadcn-vue is initialized in internal/features/dashboard/ui (components.json, Tailwind config, CSS design tokens) with a DARK theme matching the current mission-control palette (blue accent); base components added (button, card, badge, table, progress, separator, tooltip); npm run build still produces a single self-contained dist/index.html (vite-plugin-singlefile preserved), no runtime CDN
- [ ] npm run build + npm run test:unit + go build ./... stay green; the app still renders (even if sections not yet refactored)
## Log
