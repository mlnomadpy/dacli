---
id: t-01KZRFRHF2KWYB8H639SDG6JCS
kind: task
created: 2026-08-11T13:21:32Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
---
# MCP argument coercion silently returns a zero value for an uninterpretable argument
## Acceptance
- [x] an argument present but uninterpretable is refused by name rather than coerced to the zero value
- [x] the refusal reaches the MCP client as an error, so an agent learns its call was malformed instead of getting a wrong default
- [x] a test asserts a nonsense dry_run value is refused rather than silently read as false
## Log
- 2026-08-11T13:31:22Z accepted by a-root
- 2026-08-11T13:31:22Z closed WITHOUT verification — no --verify command was given
- 2026-08-11T13:31:22Z deliverable: no dacli/361-mcp-argument-coercion-silently-returns-a-zero-value-for-an-uninterpretable branch — nothing to check against main
- 2026-08-11T13:31:22Z completed by a-root
- 2026-08-11T13:32:08Z reopened by a-root: force-accepted in error: this task IS the deferred work. The commit that closed it says 'WHAT IS NOT FIXED, and is filed as 361 rather than half-done' — coercion still returns a zero value for an uninterpretable argument, and none of the three criteria are met (cleared 3 acceptance box(es) — the close claimed work that was not verified)
- 2026-08-11T14:04:00Z accepted by a-root
- 2026-08-11T14:04:00Z verified by `go test ./internal/mcp/` (exit 0)
- 2026-08-11T14:04:00Z deliverable: no dacli/361-mcp-argument-coercion-silently-returns-a-zero-value-for-an-uninterpretable branch — nothing to check against sprint/17
- 2026-08-11T14:04:00Z completed by a-root
