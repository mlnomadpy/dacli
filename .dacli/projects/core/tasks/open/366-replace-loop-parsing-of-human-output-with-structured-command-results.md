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
- [ ] Loop obtains spawned run IDs through a typed or machine-readable result rather than parsing the spawn banner
- [ ] Loop obtains ship integration counts through a typed or machine-readable result rather than parsing human prose
- [ ] Tests change the human formatter text while proving orchestration behavior remains unchanged
- [ ] No loop control decision scans user-facing output for run IDs or integrated branch counts
- [ ] go test -race ./... passes
## Log
- 2026-08-12T19:44:08Z claimed by a-codex-maintainer-xscvft
