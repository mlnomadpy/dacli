---
id: t-01KYFQ5HRQZ0FY3MZC3B99WPDZ
kind: task
created: 2026-07-26T17:22:07Z
created_by: a-hxr220kqc4
owner: a-root
priority: must
github:
  issue: 250
  repo: mlnomadpy/dacli
---
# CI never runs the dashboard SPA vitest suite — wire npm run test:unit (and lint) into ci.yml so frontend regressions fail the build
## So that
the 14 frontend test files that lock in shipped fixes (DependencyGraph.test.ts for task 150's critical-edge, BurnRate.test.ts for task 149's yell) actually guard against regressions instead of never executing in CI
## Acceptance
- [x] ci.yml 'test' job runs 'npm run test:unit' (vitest) in internal/features/dashboard/ui after the frontend build, and a failing frontend test fails the CI job
- [x] ci.yml also enforces the frontend lint config: runs 'npm run lint' (eslint --max-warnings 0), which today is defined in package.json but never invoked by any workflow
- [x] verified by intentionally breaking one component assertion and confirming the CI test job goes red (not green)
- [x] no duplication of the go test step; the frontend test step is distinct and runs regardless of Go test outcome ordering
## Log
- 2026-07-26T20:53:12Z claimed by a-azfs9hw109
- 2026-07-26T20:58:38Z adopted by a-root (owner a-hxr220kqc4 orphaned)
- 2026-07-26T20:58:38Z accepted by a-root
- 2026-07-26T20:58:38Z completed by a-root
