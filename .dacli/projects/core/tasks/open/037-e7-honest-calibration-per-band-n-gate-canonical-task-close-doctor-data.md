---
id: t-01KY55NX8EVAR400DQ3VF5AR9X
kind: task
created: 2026-07-22T15:04:04Z
created_by: a-root
owner: a-root
priority: should
estimate: {optimistic: 2, probable: 4, pessimistic: 6}
---
# E7: honest calibration — per-band n-gate, canonical task-close, doctor data-integrity check
## Acceptance
- [ ] by-agent (and size) bands with n<10 are marked provisional and do not print a p10-p90 range as if calibrated
- [ ] accept and task done share one close primitive that always stamps 'completed by'; no path can close a task without the actuals stamp
- [ ] dacli doctor flags done tasks with a claim but no completion stamp (broken calibration spans)
- [ ] committed on branch by an agent; build + test green
## Log
