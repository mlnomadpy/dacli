---
id: f-050-complete-on-branch-dacli-050-fix-ship-vcs-half-ship-guard-integrate-error
kind: note
note_kind: finding
created: 2026-07-22T16:29:31Z
created_by: a-jjwx3z556n
about: [[050]]
severity: major
---
# 050 complete on branch dacli/050-fix-ship-vcs: half-ship guard + integrate error propagation + cross-project refs, commit 13d502d
Commit 13d502d by a-jjwx3z556n (maintainer), 5 files (git add + dacli commit --no-add --force; gitx is the shared layer the vcs merge fix lives in — see decision note): internal/gitx/{gitx.go,gitx_test.go}, internal/features/vcs/lifecycle.go, internal/features/ship/{ship.go,ship_test.go}. ACCEPTANCE all met: (1) NEVER HALF-SHIPS — gitx.Merge now guards with IsCleanExcept(root,'.dacli') so accept's tracked task-file moves no longer make the merge no-op (the happy path works); a GENUINE non-conflict integrate failure now propagates a non-zero exit, so ship stops at integrate BEFORE commit/push (TestShipStopsOnIntegrateError); and the record commit message reports branches ACTUALLY merged, parsed from integrate's 'integrated N branch(es)' line via integratedCount(), not len(done) (TestShipRecordMessageReportsActualMerges). (2) cmdIntegrate (lifecycle.go:271) distinguishes a conflict-block (clikit.ExitCode==3, exit 0, task blocked, ship detects semantically) from a hard error (dirty code, missing branch, unrelated histories) which it now returns as a non-zero error instead of mislabelling 'conflict' and swallowing to nil — regression of the 018 fix closed (TestMergeMissingBranchIsErrorNotConflict). (3) doneRefs emits each task's globally-unique ULID (fallback %03d-slug) so a multi-project done list no longer collapses to an ambiguous bare seq and integrate resolves each ref to exactly one task (TestDoneRefsQualifiesAcrossProjects). (4) committed by an agent; go build ./... clean; go test ./internal/... all green (15 pkgs, incl. new gitx_test.go + ship tests, TestFeatureSlicesAreIsolated); go vet + gofmt clean. Box-checking refused for non-owner (only a-root) — owner: verify and close via dacli task check/done + dacli merge --task 050.
