---
id: t-01KZ65YW5FX94R16J63C453J9K
kind: task
created: 2026-08-04T10:43:54Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 0.5, probable: 1.5, pessimistic: 4}"
---
# A role can declare grant rw and a read-only runtime, and nothing catches it
## So that
a spawn that cannot possibly write is refused at spawn time instead of burning a run
## Acceptance
- [x] spawn refuses when the role grant is rw but the runtime has no write tool
- [x] doctor reports every role whose grant and runtime capability disagree
- [x] junior is either given a write-capable runtime or its grant is corrected
## Log
- 2026-08-04T11:35:31Z claimed by a-maintainer-0fh816
- 2026-08-04T11:51:05Z accepted by a-root
- 2026-08-04T11:51:05Z verified by `go test ./internal/store/ ./internal/features/execution/ ./internal/features/insight/` (exit 0)
- 2026-08-04T11:51:05Z completed by a-root
- 2026-08-04T18:18:12Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/310 (event 01KZ69TD2H8S56KN75ZN1EECCX)
