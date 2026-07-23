---
id: f-task-add-now-refuses-near-duplicate-titles-vs-the-open-active-backlog
kind: note
note_kind: finding
created: 2026-07-23T22:49:27Z
created_by: a-wgghcfe1sf
about: [[116]]
severity: moderate
---
# task add now refuses near-duplicate titles vs the open/active backlog
store.TitleSimilarity (internal/store/similarity.go) is a Jaccard index over normalized, crudely-stemmed, stopword-filtered title tokens, gated by a minSharedTitleTokens=2 floor so two short titles sharing one generic word (e.g. worktree test fixtures 'Feature A' / 'Feature B') never collide. store.FindNearDuplicateTask checks a new title against StatusOpen+StatusActive tasks in the project; cmdTaskAdd (internal/features/planning/planning.go) refuses (clikit.Refusedf, exit 3) when the best match scores >= store.DuplicateTitleThreshold (0.5), unless --force is passed. Threshold calibrated against the exact pairs reported in task 116: 106/108 scores 0.75, 109/110 scores 0.5. go build ./... clean; go test ./internal/... all green (caught and fixed one false positive during development: internal/cli TestParallelWorktreeLifecycle's 'Feature A'/'Feature B' fixture titles, fixed via the minSharedTitleTokens gate).
