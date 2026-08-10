---
id: t-01KZPQMCBY2XJWR9BPXXGQYG0Z
kind: task
created: 2026-08-10T21:00:36Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 1, probable: 2, pessimistic: 3}"
---
# Derived claims must resolve to real repo paths, or a claim locks an agent out of its own files
## Acceptance
- [x] a task whose prose contains a slash but names no real path yields no claim, so the agent is warned rather than blocked
- [x] a bare filename that is not a repo-relative path yields no claim, because claims match by exact path or prefix and would overlap nothing staged
- [x] a task naming a real path still yields that claim, so the coordination the claim exists for is not lost
- [x] routing keeps using the crude PathHints, where a spurious token costs one tie-break vote rather than a lockout
## Log
- 2026-08-10T21:01:16Z accepted by a-root
- 2026-08-10T21:01:16Z verified by `go test ./internal/store/ ./internal/features/orchestration/` (exit 0)
- 2026-08-10T21:01:16Z deliverable: no dacli/343-derived-claims-must-resolve-to-real-repo-paths-or-a-claim-locks-an-agent-out-of branch — nothing to check against sprint/5
- 2026-08-10T21:01:16Z completed by a-root
