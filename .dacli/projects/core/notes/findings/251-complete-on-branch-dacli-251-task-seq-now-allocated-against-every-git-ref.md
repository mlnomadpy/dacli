---
id: f-251-complete-on-branch-dacli-251-task-seq-now-allocated-against-every-git-ref
kind: note
note_kind: finding
created: 2026-08-04T11:44:37Z
created_by: a-maintainer-88hjw4
about: "[[251]]"
severity: major
---
# 251 complete on branch dacli/251-...: task seq now allocated against every git ref
Commit 1479bfd (a-maintainer-88hjw4). Root cause: store.CreateTask (internal/store/store.go:389-401) computed next seq only from ListTasks(working tree), so two branches cut from the same base each saw the same max and handed out the same NNN; on merge two different tasks share one seq and 'dacli <NNN>' fails ambiguous (cross-branch twin of 209/247). FIX (3 acceptance items): (1) new gitTaskSeqCeiling(w,project) scans 'git log --all --name-only' over .dacli/projects/<project>/tasks and raises the ceiling to the max seq committed on ANY ref, incl. renamed/deleted files (monotonic-never-reuse); best-effort, returns 0 off-git so non-git tests unchanged. CreateTask now takes max(workingTreeMax, gitCeiling)+1 inside the existing seq lock. (2) regression test TestCreateTaskNeverReusesASeqFromAnotherBranch (internal/store/task_seq_branch_test.go) builds a real 2-branch git repo and reproduces the collision; verified FAIL before the fix ('branch B reused branch A's seq 1'), PASS after. Plus deleted-branch-seq and no-repo-degradation tests. (3) pre-existing collisions reconciled via new store.CollidedSeqs + doctor 'collided-seq' check (insight.go) that names both tasks so an owner renumbers one, rather than leaving the ref silently broken. Proof: go build ./..., go vet ./..., gofmt -l internal/ clean; go test ./internal/store ./internal/features/insight green. NOTE: the only full-suite failure is internal/features/catalog TestCatalogRefuses... = the pre-existing DACLI_AGENT env-leak (catalog lacks a TestMain clearing it, unlike internal/cli), unrelated to this change. All task creation paths (task add->planning.go, ghmirror, onboard, orchestration) route through store.CreateTask, so the fix is uniform. Owner: dacli accept 251 then integrate/merge.
