---
id: t-01KZYA0JZ7EST3H42M5HTXNW21
kind: task
created: 2026-08-13T19:36:31Z
created_by: a-codex-loop-auditor-hxqjcg
owner: a-root
priority: should
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
parent: "[[t-01KZXXZPBXT332W00RP94HTR2K]]"
github:
  issue: 609
  repo: mlnomadpy/dacli
---
# Version MCP tool schemas with golden compatibility fixtures
## So that
the documented MCP compatibility promise is enforced for every stable tool
## Acceptance
- [x] every Tier-1 MCP tool schema declares a version and has a golden JSON fixture under internal/mcp/testdata
- [x] compatibility tests accept additive changes and fail on removed, renamed, or retyped stable fields
- [x] malformed and unsupported-version requests are rejected before any mutating handler runs
## Log
- 2026-08-13T21:57:00Z claimed by a-fixer-sn8j7p
- 2026-08-13T22:18:47Z adopted by a-root (owner a-codex-loop-auditor-hxqjcg orphaned)
- 2026-08-13T22:18:47Z accepted by a-root (applied 1 proposal(s))
- 2026-08-13T22:18:47Z verified by `go test ./internal/mcp ./internal/cli` (exit 0) in branch main at 12d987e — proves that tree builds, not that the work is in trunk
- 2026-08-13T22:18:47Z deliverable: dacli/435-version-mcp-tool-schemas-with-golden-compatibility-fixtures is merged into main
- 2026-08-13T22:18:47Z completed by a-root
- 2026-08-13T23:51:31Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/638 (event 01KZYJVNCTEBB3NGHFCM4C04CC)
## Verification Evidence
{"command":"go test ./internal/mcp ./internal/cli","exit_code":0,"duration_ms":36869,"artifact_hash":"sha256:fcea03206ec4a1f6a8071757d1dfcb2cb4b5929ac8e8e27494df44cbe919720e","verifier":"a-root","branch":"main","commit_sha":"12d987eed7f3bd549d7276c2660b53fd876cf443"}
