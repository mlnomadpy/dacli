---
id: 01KYFQ5TDY9D6JAEWBMGF5WST4
kind: event
event_kind: finding
created: 2026-07-26T17:22:16Z
created_by: a-hxr220kqc4
about: [[t-01KY60QM1Y7DK05WXB954YNDHJ]]
origin: agent
applied: false
---
CI builds the SPA but never runs its vitest suite — 14 frontend tests are dead weight in CI

ci.yml:24-34 runs 'npm ci && npm run build' then gofmt/vet/'go test ./...'/go build — it never runs 'npm run test:unit'. package.json:9 shows build = run-p type-check build-only (vue-tsc + vite build ONLY, no vitest); the vitest runner is a separate script, test:unit (package.json:13). So the 14 frontend test files under internal/features/dashboard/ui/src/**/__tests__/ (9 components incl. DependencyGraph.test.ts and BurnRate.test.ts, 3 composables, store, App) NEVER execute in CI. A behavioral regression in any Vue component/composable/Pinia store — including the just-shipped task-150 DAG critical-edge fix and task-149 burn-yell fix, both of which have tests — ships with CI green. eslint (package.json:15 lint) is likewise never invoked by any workflow. Task 135:15 deliberately wired only 'npm run build' into CI; task 132:14 only asserted test:unit passes LOCALLY. Filed as task 154.
