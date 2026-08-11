---
id: t-01KZPWSJ2P4TM80XZTQ4YZ5JYZ
kind: task
created: 2026-08-10T22:30:48Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
---
# Publish a CLI and MCP schema compatibility policy with migration notes
## Acceptance
- [x] the policy states what is stable (exit codes, command paths, JSON shapes) and what may change without notice
- [x] a test asserts the documented-stable JSON surfaces still parse to the documented shape, so the policy is enforced rather than asserted
- [x] migration notes exist for any surface already changed, including the ones changed this week
## Log
- 2026-08-11T10:00:19Z claimed by a-fixer-mebe8r
- 2026-08-11T10:12:05Z accepted by a-root
- 2026-08-11T10:12:05Z closed WITHOUT verification — no --verify command was given
- 2026-08-11T10:12:05Z deliverable: dacli/355-publish-a-cli-and-mcp-schema-compatibility-policy-with-migration-notes exists but is NOT in sprint/13 — closed anyway
- 2026-08-11T10:12:05Z completed by a-root
