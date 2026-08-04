---
id: f-215-complete-worktree-path-now-keyed-on-project-seq-slug-commit-3daa09a
kind: note
note_kind: finding
created: 2026-08-04T10:18:17Z
created_by: a-maintainer-mshdxy
about: "[[215]]"
severity: moderate
---
# 215 complete: worktree path now keyed on project+seq+slug (commit 3daa09a)
Branch dacli/215-worktree-path-is-keyed-on-slug-alone-so-same-titled-tasks-collide, commit 3daa09a by a-maintainer-mshdxy. ROOT CAUSE: workspace.WorktreePath(slug) keyed the worktree dir on slug alone (workspace.go:347-348), while the branch already used seq (BranchFor: dacli/%03d-%s). Two same-titled tasks share a slug, so they collided onto ONE worktree dir and committed onto the wrong branch. FIX: WorktreePath(project, seq, slug) now composes name = <project>-<NNN>-<slug> (workspace.go:347-358), guarding empty project. Updated all 5 callers to pass t.Project,t.Seq,t.Slug (vcs/lifecycle.go:115,152,931,1178; execution/execution.go:580). ACCEPTANCE 'worktree paths include the project and seq' is met. TEST: added TestWorktreePathDisambiguatesSameTitle (workspace_test.go) asserting same-slug/different-seq and same-seq/different-project do not collide and base==core-001-fix-thing; REPRODUCED the bug — with the slug-only body it fails 'same-titled tasks share a worktree', with the fix it passes. PROOF: go build ./... clean, go vet clean, gofmt -l internal/ empty, go test green for workspace/cli/vcs/execution. NOTE: internal/features/catalog fails 'agent token not recognized' but that PRE-EXISTS on clean main (DACLI_AGENT env leak; catalog lacks the TestMain clear internal/cli has) and is unrelated. Owner: dacli accept 215 then integrate/merge --task 215.
