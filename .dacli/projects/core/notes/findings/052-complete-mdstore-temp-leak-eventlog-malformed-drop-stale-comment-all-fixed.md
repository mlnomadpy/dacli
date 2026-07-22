---
id: f-052-complete-mdstore-temp-leak-eventlog-malformed-drop-stale-comment-all-fixed
kind: note
note_kind: finding
created: 2026-07-22T16:21:56Z
created_by: a-hq6ebk9c35
about: [[052]]
severity: moderate
---
# 052 complete: mdstore temp-leak, eventlog malformed-drop, stale comment all fixed
Commit c8840f1 by a-hq6ebk9c35 on branch dacli/052-fix-mdstore-eventlog-temp-file-leak-on-rename-drop-malformed-events-stale. Staged ONLY 5 files (git add + dacli commit --no-add): internal/mdstore/{mdstore.go,mdstore_test.go}, internal/eventlog/{eventlog.go,eventlog_test.go,sync.go}.

ACCEPTANCE, all satisfied:
(1) mdstore.WriteFile (mdstore.go:471) now os.Remove(name)s its temp file on the os.Rename error branch, matching the WriteString/Close error paths that already cleaned up — a cross-device/EACCES/index-lock rename fault no longer orphans a .dacli-tmp-* file. New TestWriteFileCleansTempOnRenameFailure forces rename failure (target is a non-empty dir) and asserts no temp litter.
(2) eventlog.List (eventlog.go) no longer silently drops a malformed/unreadable event: the per-file mdstore.ReadFile error now log.Printf's 'skipping unreadable event <path>: <err>' (and the WalkDir error branch logs 'walking <path>: <err>') instead of a bare continue, so a corrupt/half-written event surfaces in stderr rather than vanishing from the append-only log — while the rest of the log still lists (one bad file does not blind every reader). New TestListSurfacesMalformedEvent proves the readable event survives and the bad one is logged.
(3) sync.go:184 stale logOnce comment corrected: it claimed 't.Doc reflects the durable Log because Sync loads the task fresh each pass' — Sync no longer reloads per event; it builds one store.BuildTaskIndex up front and mutates a shared *Task pointer in place, rebuilding the index per invocation. Comment now describes the index-shared-pointer model.
(4) go build ./... clean; go test -exec 'env -u DACLI_AGENT' ./internal/... all green (incl. mdstore, eventlog, cli, TestFeatureSlicesAreIsolated).

Box-checking refused for non-owner (only a-root). Owner: verify and close via dacli task check/done + dacli merge --task 052.
