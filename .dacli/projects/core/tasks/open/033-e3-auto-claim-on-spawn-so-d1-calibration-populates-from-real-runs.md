---
id: t-01KY53QHGE1D1EJ09FXG8733K4
kind: task
created: 2026-07-22T14:30:00Z
created_by: a-root
owner: a-root
priority: should
estimate: {optimistic: 2, probable: 3, pessimistic: 5}
---
# E3: auto-claim on spawn so D1 calibration populates from real runs
## Acceptance
- [ ] spawn stamps a claim on the task at launch so a claim->done span exists; calibrate by-agent-band then joins run records to actuals and stops being empty on real agent data
- [ ] no double-claim on re-spawn/supervise; existing claim is respected
- [ ] committed on branch by an agent; build + test green
## Log
