---
id: t-01KZ78NAS4P0NZXW4ARVZZ52T3
kind: task
created: 2026-08-04T20:50:22Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 2, probable: 4, pessimistic: 8}"
---
# Audit the loop end to end against every capability shipped today
## So that
the unattended path is the best-covered one rather than the least, since it is the one nobody is watching
## Acceptance
- [x] each of the six phases is traced against what it actually calls, and any capability the loop bypasses is named
- [x] findings state what an unattended run would do wrong, not merely what it omits
- [x] anything already filed is reported as already filed
## Log
- 2026-08-09T23:12:21Z accepted by a-root
- 2026-08-09T23:12:21Z verified by `go build ./...` (exit 0)
- 2026-08-09T23:12:21Z deliverable: no dacli/300-audit-the-loop-end-to-end-against-every-capability-shipped-today branch — nothing to check against trunk
- 2026-08-09T23:12:21Z completed by a-root
