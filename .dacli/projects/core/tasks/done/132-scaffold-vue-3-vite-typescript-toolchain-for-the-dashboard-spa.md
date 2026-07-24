---
id: t-01KY8KW3WK0EHPC7Y4FZCFMCES
kind: task
created: 2026-07-23T23:09:51Z
created_by: a-root
owner: a-root
priority: must
depends_on: [131]
---
# Scaffold Vue 3 + Vite + TypeScript toolchain for the dashboard SPA
## So that
the SPA has a modern, tested, lint-clean foundation
## Acceptance
- [x] internal/features/dashboard/ui/ holds a Vue 3 + Vite + TS project using <script setup> Composition API, Pinia, ESLint + Prettier, and Vitest; npm ci && npm run build && npm run test:unit all pass
- [x] npm run build outputs static assets to ui/dist for Go to embed; zero runtime external-CDN dependencies
## Log
- 2026-07-23T23:20:51Z claimed by a-cs8yn6q0z2
- 2026-07-24T00:12:08Z accepted by a-root
- 2026-07-24T00:12:08Z completed by a-root
