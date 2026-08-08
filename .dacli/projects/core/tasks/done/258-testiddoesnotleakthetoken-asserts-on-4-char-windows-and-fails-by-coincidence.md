---
id: t-01KZ63T6SPC4TP4RC9F8WGWAPG
kind: task
created: 2026-08-04T10:06:24Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 0.2, probable: 0.3, pessimistic: 0.6}"
---
# TestIDDoesNotLeakTheToken asserts on 4-char windows and fails by coincidence
## So that
CI on every PR stops going red for a reason that is not a defect
## Acceptance
- [x] the assertion window is the discriminator length, so a derived discriminator still trips it
- [x] 200 consecutive runs of the test are green
## Log
- 2026-08-04T10:06:31Z claimed by a-root
- 2026-08-04T10:07:22Z completed by a-root
- 2026-08-04T18:18:12Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/284 (event 01KZ63XNFZMWDR0XHM4YP79S7S)
