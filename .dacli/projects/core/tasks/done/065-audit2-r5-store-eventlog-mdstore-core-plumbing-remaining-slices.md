---
id: t-01KY5GP5SE357058NVZKAG0WJ5
kind: task
created: 2026-07-22T18:16:27Z
created_by: a-root
owner: a-root
priority: should
---
# AUDIT2 R5: store + eventlog + mdstore + core plumbing + remaining slices
## Acceptance
- [x] findings (file:line) in internal/{store,eventlog,mdstore,prompts,workspace,clikit,gitx,skills} + remaining feature slices. Focus on code changed since the last audit (G-series ghmirror, F1/F2 calibration+stream-json, ship/accept slices, G3 pr) and VERIFY the previous audit's fixes held (no regressions). File each issue as a finding with file:line, severity-tagged. Read-only — do not edit.
## Log
- 2026-07-22T18:16:35Z claimed by a-cw27djtx7d
- 2026-07-22T18:23:33Z finding by a-cw27djtx7d: accept-propose events are consumed by eventlog.Sync, silently dropping an agent's acceptance proposal (event 01KY5GXWM33NA3VR8E7Y6Q5GDC)
- 2026-07-22T18:23:33Z finding by a-cw27djtx7d: ghmirror findingAboutTask matches a task's zero-padded seq as a bare substring, mis-attributing findings across tasks (event 01KY5GY5EXMRHP90KG8EJG8ZE2)
- 2026-07-22T18:23:33Z finding by a-cw27djtx7d: R5 audit: prior plumbing fixes verified held in current tree (store/eventlog/mdstore/gitx/calibration/taint/prompts/ship) (event 01KY5GYW35T8BFVV4RATMN4YDG)
- 2026-07-22T18:52:27Z accepted by a-root
- 2026-07-22T18:52:27Z completed by a-root
