---
id: f-loop-build-phase-now-ranks-ready-frontier-by-moscow-priority-cpm-slack-before
kind: note
note_kind: finding
created: 2026-07-23T13:38:29Z
created_by: a-hkm1s8wvp9
about: [[103]]
severity: moderate
---
# loop BUILD phase now ranks ready frontier by MoSCoW priority + CPM slack before slicing to width
readyTasks() in internal/features/orchestration/orchestration.go still returns tasks in Seq order (unchanged), but loop() now calls the new rankByPriority(w, project, ready) right after readyTasks (orchestration.go loop(), around line 286-290) before gov.Before/runCycle slice to width. rankByPriority sorts by model.Priority(...).Rank() first, then CPM slack from the new criticalPathSlack() helper (a duplicate of insight.cmdNext's CPM block per the feature-slice isolation rule — orchestration cannot import the insight feature slice), then Seq as the final tiebreak — mirroring cmdNext (internal/features/insight/insight.go:210-219) exactly so `dacli next` and the loop's BUILD phase agree on selection order.
