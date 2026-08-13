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
- [x] DESIGN.md no longer describes implemented runtime adapters as specification-only
- [x] docs/RUNTIMES.md lists only shipped presets or clearly labels examples that require user configuration
- [x] docs/COMPATIBILITY.md and docs/MCP.md accurately describe the manually maintained MCP tool surface
- [x] README.md, docs/SPM.md, and SECURITY.md contain no unsupported lint flag or stale unreleased-state claim
- [x] A repository check or focused test fails when the corrected support claims drift again
## Log
- 2026-08-12T20:09:01Z claimed by a-codex-maintainer-p44wb5
- 2026-08-12T20:24:23Z accepted by a-root
- 2026-08-12T20:24:23Z verified by `go test ./docs -run TestPublicSupportClaimsMatchShippedSurface -count=1` (exit 0) in branch main at 6fc9e5d — proves that tree builds, not that the work is in trunk
- 2026-08-12T20:24:23Z deliverable: dacli/367-reconcile-runtime-mcp-and-release-documentation-with-shipped-behavior is merged into main
- 2026-08-12T20:24:23Z completed by a-root
- 2026-08-12T20:40:43Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/537 (event 01KZVSXEZFZWERYXH6SJ8ADR8R)
- 2026-08-12T20:40:43Z a-root: Integrated via PR https://github.com/mlnomadpy/dacli/pull/537 at merge commit 6fc9e5d779d2fe950e6bddf7e5fbc7444333e866 into main; local cleanup complete (event 01KZVT8KQH5EPWXDHASM3A06CM)
