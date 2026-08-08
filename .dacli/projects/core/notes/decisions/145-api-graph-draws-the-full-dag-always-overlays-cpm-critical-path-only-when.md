---
id: d-145-api-graph-draws-the-full-dag-always-overlays-cpm-critical-path-only-when
kind: note
note_kind: decision
created: 2026-07-24T11:23:13Z
created_by: a-gbyc86v99b
about: [[145]]
---
# 145: /api/graph draws the full DAG always, overlays CPM critical path only when the open subset is schedulable (degrades with a note, never refuses)
## Chose
145: /api/graph draws the full DAG always, overlays CPM critical path only when the open subset is schedulable (degrades with a note, never refuses)
## Rejected
mirroring cmdCriticalPath's hard refusal on any unestimated/cyclic open task
## Because
the DAG view's job is to always show the dependency chain (nodes=all tasks by status, edges=depends_on); refusing the whole surface because one open task lacks an estimate would blank the operator's map. buildGraph draws every node+edge unconditionally and only the critical-path highlight degrades to graphView.Note, matching the codebase's honest-degrade ethos (buildBurn) rather than critical-path's exit-refusal.
