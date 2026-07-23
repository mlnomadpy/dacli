---
id: t-01KY757X1YYPGTJB2VTGN0YT8A
kind: task
created: 2026-07-23T09:34:54Z
created_by: a-root
owner: a-root
priority: must
---
# Wire real per-cycle token accounting into the loop so --window-tokens enforces
## Acceptance
- [x] runCycle sums the cycle's spawn usage.txt token actuals and returns them; the Governor's token window is no longer a no-op
- [x] A test asserts the token-window governor sleeps once the window budget is exceeded with real per-cycle tokens
## Log
- 2026-07-23T10:20:12Z claimed by a-s56ztpjbv1
- 2026-07-23T10:24:07Z accepted by a-root
- 2026-07-23T10:24:07Z completed by a-root
