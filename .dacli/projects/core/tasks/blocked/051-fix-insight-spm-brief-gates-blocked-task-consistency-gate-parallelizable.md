---
id: t-01KY59YW0RM6K6H2GA2XX4WRJE
kind: task
created: 2026-07-22T16:18:52Z
created_by: a-root
owner: a-root
priority: should
estimate: {optimistic: 3, probable: 5, pessimistic: 8}
---
# FIX insight/spm/brief/gates: blocked-task consistency, gate, parallelizable, MillerCap, calibrate perf
## Acceptance
- [ ] next and critical-path agree on blocked tasks (both exclude, or both include-with-flag) — no error when an open task depends on a blocked one
- [ ] decisions gate verifies a rejection exists (matching its description) rather than only counting notes; spm Network.Parallelizable does not claim dependency-satisfied filtering it cannot perform
- [ ] brief 'What siblings found' honors MillerCap like every other section; calibrate walks RunsDir once (not 2-3x) per readout
- [ ] committed by an agent; go build + go test ./internal/... green
## Log
- 2026-07-22T16:19:09Z claimed by a-8g6b17xcdq
- 2026-07-22T16:37:22Z blocked on merge conflict
