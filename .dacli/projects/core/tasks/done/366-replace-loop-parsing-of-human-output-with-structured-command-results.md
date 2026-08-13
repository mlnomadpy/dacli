---
id: t-01KZV16DPQY7E2C1JMEJ08DQVM
kind: task
created: 2026-08-12T13:04:42Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
depends_on: [371, 373]
github:
  issue: 459
  repo: mlnomadpy/dacli
---
# Replace loop parsing of human output with structured command results
## So that
Autonomous orchestration does not depend on intentionally unstable presentation strings
## Acceptance
- [x] Loop obtains spawned run IDs through a typed or machine-readable result rather than parsing the spawn banner
- [x] Loop obtains ship integration counts through a typed or machine-readable result rather than parsing human prose
- [x] Tests change the human formatter text while proving orchestration behavior remains unchanged
- [x] No loop control decision scans user-facing output for run IDs or integrated branch counts
- [x] go test -race ./... passes
## Log
- 2026-08-12T19:44:08Z claimed by a-codex-maintainer-xscvft
- 2026-08-12T19:56:48Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/531 (event 01KZVRRVFYXSHGV47X26RVVKAS)
- 2026-08-12T20:07:23Z accepted by a-root
- 2026-08-12T20:07:23Z verified by `go test -race ./...` (exit 0) in branch main at dee0d91 — proves that tree builds, not that the work is in trunk
- 2026-08-12T20:07:23Z deliverable: dacli/366-replace-loop-parsing-of-human-output-with-structured-command-results is merged into main
- 2026-08-12T20:07:23Z completed by a-root
- 2026-08-12T20:40:43Z a-root: Integrated via PR https://github.com/mlnomadpy/dacli/pull/531 at merge commit dee0d91e77b86e8f4f5da2e8b43ec03795d41275 into main; local cleanup complete (event 01KZVS3ZQ4EJ9BP3FMTD30FXAR)
