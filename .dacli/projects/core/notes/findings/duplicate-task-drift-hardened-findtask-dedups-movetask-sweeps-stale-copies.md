---
id: f-duplicate-task-drift-hardened-findtask-dedups-movetask-sweeps-stale-copies
kind: note
note_kind: finding
created: 2026-07-22T15:34:36Z
created_by: a-7zg8j1n976
about: [[038]]
severity: major
---
# Duplicate-task drift hardened: FindTask dedups, MoveTask sweeps stale copies, doctor flags them
store.go: ListTasks now funnels through listTasksRaw+dedupeTasks — a task file in two status folders yields ONCE, keeping the most-terminal copy (statusRank over model.AllStatuses, mtime tie-break), so FindTask('26') resolves cleanly to the done copy instead of erroring 'ambiguous' on the same task twice (store.go:321,428). MoveTask (store.go:~509) now sweeps every other status folder for a same-named copy after the rename, so a pre-existing stale copy can never survive a move. New store.DuplicateTaskFiles walks the raw duplicated view; insight.go cmdDoctor (~:824) reports 'duplicate-task-file' naming the paths. Tests: store_dedup_test.go (TestFindTaskDedupsStaleDuplicate, TestMoveTaskLeavesNoStaleCopy, TestDuplicateTaskFilesReportsDrift). go build + go test ./internal/... green.
