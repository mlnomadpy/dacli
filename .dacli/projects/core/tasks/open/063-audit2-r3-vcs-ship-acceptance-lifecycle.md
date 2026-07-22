---
id: t-01KY5GP5RHAPGVT35FDFS4Z0PB
kind: task
created: 2026-07-22T18:16:27Z
created_by: a-root
owner: a-root
priority: should
---
# AUDIT2 R3: vcs + ship + acceptance lifecycle
## Acceptance
- [ ] findings (file:line) in internal/features/{vcs,ship,acceptance}/**. Focus on code changed since the last audit (G-series ghmirror, F1/F2 calibration+stream-json, ship/accept slices, G3 pr) and VERIFY the previous audit's fixes held (no regressions). File each issue as a finding with file:line, severity-tagged. Read-only — do not edit.
## Log
- 2026-07-22T18:16:35Z claimed by a-7sy0x8b84g
- 2026-07-22T18:23:33Z finding by a-7sy0x8b84g: cmdPR records the PR URL as an EventFinding, permanently dragging the task's brief trust-floor to unverified (event 01KY5H0N6VK88MZM2WR43QTYWR)
- 2026-07-22T18:23:33Z finding by a-7sy0x8b84g: cmdPR has no rw-grant check, so a read-only agent can publish a PR (and internal findings/verdicts) to GitHub (event 01KY5H0YJTAP17YB4W4K3410BM)
- 2026-07-22T18:23:33Z finding by a-7sy0x8b84g: cmdContrib double-counts a findings-against filed by a read-only reviewer (once as the event, again as its synced note) (event 01KY5H17X1ZJWNZKQSK0J80701)
- 2026-07-22T18:23:33Z finding by a-7sy0x8b84g: AUDIT2 R3 regression check: prior audit fixes in vcs/ship/acceptance/gitx all held (event 01KY5H1JXEF2G8Z0YNHXAC5BDC)
