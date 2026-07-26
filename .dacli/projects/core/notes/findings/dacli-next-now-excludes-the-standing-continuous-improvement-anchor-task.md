---
id: f-dacli-next-now-excludes-the-standing-continuous-improvement-anchor-task
kind: note
note_kind: finding
created: 2026-07-26T23:00:42Z
created_by: a-fw66q78xzb
about: [[112]]
severity: minor
---
# dacli next now excludes the standing Continuous improvement anchor task
insight.go cmdNext (internal/features/insight/insight.go) built its open-tasks slice from every non-done, non-blocked task, so the loop's review-phase anchor (title prefix 'Continuous improvement', filed by orchestration.go ensureImproveTask) could surface as the #1 MoSCoW recommendation once MUST-priority work ran out — confirmed reproducible before the fix. Fix: added store.Task.IsLoopAnchor() + store.ContinuousImprovementMarker (internal/store/store.go) as the single shared predicate, used by both readyTasks (internal/features/orchestration/orchestration.go:826, previously an inline strings.HasPrefix check) and cmdNext's open-task filter (internal/features/insight/insight.go:151-158, new exclusion). Covered by new tests TestNextSkipsContinuousImprovementAnchor and TestNextWithOnlyAnchorOpenReportsNoneReady in internal/features/insight/insight_test.go. go build ./... clean; go test ./internal/... all green.
