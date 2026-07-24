---
id: t-01KY9W5QWZ8B7K0TRDTXKZ9FN9
kind: task
created: 2026-07-24T10:54:09Z
created_by: a-root
owner: a-root
priority: must
---
# Dashboard: task dependency / DAG view (research shortlist #3, RICE 3.2)
## So that
the operator stops manually reconstructing the dependency chain daily
## Acceptance
- [x] A /api/graph endpoint returns the task dependency DAG + critical path (internal/spm/criticalpath.go already computes the chain — this exposes+draws it); handler test
- [x] A Vue component draws the DAG (nodes=tasks by status, edges=depends_on, critical path highlighted); readable at 10-40 tasks; wired into the dashboard; component test
## Log
- 2026-07-24T11:17:16Z claimed by a-gbyc86v99b
- 2026-07-24T11:30:55Z accepted by a-root
- 2026-07-24T11:30:55Z completed by a-root
