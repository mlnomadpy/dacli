---
id: t-01KY5JSH4Q232ZSDAPWAY8B996
kind: task
created: 2026-07-22T18:53:14Z
created_by: a-root
owner: a-root
priority: must
estimate: {optimistic: 2, probable: 4, pessimistic: 6}
---
# FIX-cal: calibration band join — invocation records role/model, runRecords no-clobber
## Acceptance
- [x] supervise and verify write role AND model into invocation.txt (they omit them today), so their run bands match the OrDash band the calibrate gate/advise use
- [x] runRecords no longer clobbers a task's calibrated agent-band with a newer verify/supervise run's EMPTY band (keep the band that has role/model; do not overwrite with empty)
- [x] committed by an agent; go build + go test ./internal/... green
## Log
- 2026-07-22T18:53:33Z claimed by a-4cq9mr2nrj
- 2026-07-22T19:14:06Z accepted by a-root
- 2026-07-22T19:14:06Z completed by a-root
