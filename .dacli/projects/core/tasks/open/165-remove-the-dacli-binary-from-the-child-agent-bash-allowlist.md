---
id: t-01KYHWEB361M0TRXG1H8G19DY5
kind: task
created: 2026-07-27T13:32:47Z
created_by: a-root
owner: a-root
priority: must
---
# Remove the dacli binary from the child agent Bash allowlist
## So that
an agent with Write cannot overwrite dacli and have it executed
## Acceptance
- [ ] runtimes do not allowlist the writable build path
## Log
