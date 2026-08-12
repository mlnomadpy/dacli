---
id: t-01KZV16E6JQKG81D7F56SJED15
kind: task
created: 2026-08-12T13:04:42Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
depends_on: [366, 368, 371, 373]
github:
  issue: 460
  repo: mlnomadpy/dacli
---
# Reconcile runtime MCP and release documentation with shipped behavior
## So that
Operators can trust the documented support matrix and extension model
## Acceptance
- [ ] DESIGN.md no longer describes implemented runtime adapters as specification-only
- [ ] docs/RUNTIMES.md lists only shipped presets or clearly labels examples that require user configuration
- [ ] docs/COMPATIBILITY.md and docs/MCP.md accurately describe the manually maintained MCP tool surface
- [ ] README.md, docs/SPM.md, and SECURITY.md contain no unsupported lint flag or stale unreleased-state claim
- [ ] A repository check or focused test fails when the corrected support claims drift again
## Log
