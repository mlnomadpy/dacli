---
id: t-01KYHWEB361M0TRXG1H8G19DY5
kind: task
created: 2026-07-27T13:32:47Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 0.2, probable: 0.5, pessimistic: 1.5}"
---
# Remove the dacli binary from the child agent Bash allowlist
## So that
an agent with Write cannot overwrite dacli and have it executed
## Acceptance
- [x] runtimes do not allowlist the writable build path
## Log
- 2026-08-04T00:29:23Z claimed by a-junior-ecjkhs
- 2026-08-04T12:37:17Z accepted by a-root
- 2026-08-04T12:37:17Z verified by `grep -L 'Documents/GitHub/dacli/dacli' .dacli/runtimes/cc.md .dacli/runtimes/cc-rw.md .dacli/runtimes/cc-fe.md | wc -l | grep -q 3` (exit 0)
- 2026-08-04T12:37:17Z completed by a-root
