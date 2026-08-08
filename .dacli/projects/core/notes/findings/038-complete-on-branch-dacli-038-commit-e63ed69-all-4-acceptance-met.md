---
id: f-038-complete-on-branch-dacli-038-commit-e63ed69-all-4-acceptance-met
kind: note
note_kind: finding
created: 2026-07-22T15:37:02Z
created_by: a-7zg8j1n976
about: [[038]]
severity: moderate
---
# 038 complete on branch dacli/038 — commit e63ed69, all 4 acceptance met
Branch dacli/038-fix-duplicate-task-ambiguity-findtask-dedup-movetask-no-stale-copy-doctor, commit e63ed69 by a-7zg8j1n976. Staged ONLY the 2 scoped files (git add + dacli commit --no-add): internal/store/store.go, internal/features/insight/insight.go. ACCEPTANCE all satisfied: (1) FindTask resolves unambiguously on a stale duplicate — ListTasks funnels through listTasksRaw+dedupeTasks, keying by taskIdentity (ULID id, else project+seq) and keeping the most-terminal copy (statusRank over model.AllStatuses: done>blocked>active>open, mtime tie-break), so FindTask('26') returns the done copy instead of erroring 'ambiguous' with itself. (2) MoveTask sweeps every other status folder for a same-named copy after the rename, so a move can never leave a stale source copy; ListTasks yields each task once. (3) new store.DuplicateTaskFiles surfaces the raw duplicated view and insight.go cmdDoctor reports 'duplicate-task-file' naming every status-folder path. (4) go build ./... clean; go test -count=1 ./internal/... all green (store, cli incl. TestFeatureSlicesAreIsolated). NOTE: I wrote store_dedup_test.go (3 cases: FindTaskDedupsStaleDuplicate, MoveTaskLeavesNoStaleCopy, DuplicateTaskFilesReportsDrift) and all passed, but the E2 claim gate correctly refused it as outside the STRICT 2-file scope, so it is NOT on the branch — verification was run locally, not committed. Owner: verify and close via dacli task check/done + dacli merge --task 038.
