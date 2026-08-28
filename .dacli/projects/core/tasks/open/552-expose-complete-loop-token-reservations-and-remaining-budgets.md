---
id: t-01M1493JDJSHRPAE5QWE87CEWD
kind: task
created: 2026-08-28T13:31:49Z
created_by: a-root
owner: a-root
github:
  issue: 877
  repo: mlnomadpy/dacli
estimate: "{optimistic: 5, probable: 8, pessimistic: 13}"
depends_on: "[t-01M147RA4B2C2NAH855VBNT2SJ, t-01M146BA07Z5BTS3TTB2ADW7D4]"
---
# Expose complete loop token reservations and remaining budgets
## Context
Adopted from GitHub issue #877.

## Parent

Extracted from #871. Coordinate with complete-cycle preflight in #867 and resumable loop state in #859.

## Objective

Make cost-aware autonomy explain its usable budget, reservations, and recovery headroom before and during every bounded cycle.



## Non-goals

- Claiming exact billing data when a provider does not expose it.
- Unbounded autonomous spending.
- Cross-provider fallback outside explicit hybrid policy.

## Manual workaround today

Operators infer usable headroom from cycle/window settings and manually reserve enough capacity for review, correction, and integration.

## Acceptance
- [ ] `loop status --json` reports cycle limit/spent/remaining, rolling-window limit/spent/remaining/reset time, per-live-run reservation, review reservation, integration/recovery reserve, and unallocated amount with units and observed times.
- [ ] Preview and execution use the same reservation calculation; a wave is shrunk or refused before spawn when implementation plus required review/landing reserve cannot fit.
- [ ] Completed, failed, killed, and timed-out runs release or settle reservations idempotently from durable evidence.
- [ ] Unknown provider usage or advisory-only runtime limits are explicit and never rendered as enforceable remaining spend.
- [ ] Model routing explains the marginal estimate and why a cheaper capable model was selected or rejected without switching the pinned harness.
- [ ] Restart reconstructs reservations without double-counting and preserves STOP/cooldown/window state.
- [ ] Fixtures cover concurrent runs, correction turns, pending landing, crash/restart, window rollover, unknown usage, and insufficient recovery reserve.
- [ ] Mutation tests fail when reserved tokens are presented as freely available or preview/live accounting diverge.
## Log
- 2026-08-28T13:32:48Z dependency edit by a-root (event 01M1495CWDXHBTBEMSZB24BCH2)
