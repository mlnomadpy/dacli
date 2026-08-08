---
id: f-doctor-now-flags-orphaned-open-active-tasks-owner-has-no-live-process-via-new
kind: note
note_kind: finding
created: 2026-07-23T12:06:46Z
created_by: a-vrppnfvawm
about: [[095]]
severity: minor
---
# doctor now flags orphaned open/active tasks (owner has no live process) via new orphaned-task pattern
internal/features/insight/insight.go: cmdDoctor's per-task loop now checks, for every open/active task, whether Owner() is non-empty, non-root, and store.OwnerHasLiveRun(w, owner) is false (memoized per owner in a local map since tasks often share one owner). When true it reports pattern 'orphaned-task' naming each task and suggesting 'dacli accept --force (or --all --force)'. New helper store.OwnerHasLiveRun (internal/store/store.go, after DuplicateTaskFiles) scans w.RunsDir()'s proc.txt records for one whose Child==owner and procmon.AliveRecord(rec) is true, mirroring the existing agentClaims scan in internal/features/vcs/vcs.go:184-206. Distinguishes from accept's existing --force gate (acceptance.go:73-86), which triggers on CanMutate failure alone (not-me) regardless of liveness -- doctor's new check is strictly the process-liveness subset of that, so it never contradicts accept's behavior, only advises when to reach for --force. Tests: internal/features/insight/insight_test.go (new file) -- TestDoctorFlagsOrphanedTask builds a task via store.CreateTask(w, "a-deadchild", ...) (no proc.txt ever recorded for that id, so it is orphaned by construction) and asserts stdout contains both 'orphaned-task' and 'accept --force'; TestDoctorSkipsRootOwnedTask asserts a root-owned task is never flagged. go build ./... clean; go test ./... all green including internal/cli (arch isolation) and internal/store.
