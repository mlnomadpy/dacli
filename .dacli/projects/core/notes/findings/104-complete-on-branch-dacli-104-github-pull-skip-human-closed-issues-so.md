---
id: f-104-complete-on-branch-dacli-104-github-pull-skip-human-closed-issues-so
kind: note
note_kind: finding
created: 2026-07-23T13:46:03Z
created_by: a-8e1kfwnk72
about: [[104]]
severity: moderate
---
# 104 complete on branch dacli/104-github-pull-skip-human-closed-issues-so-inbound-sync-doesn-t-resurrect-settled
Commit 7faa5b4 by a-8e1kfwnk72 (fixer). shouldImport (internal/features/ghmirror/ghmirror.go:375-388) now returns false for an unmapped issue whose State EqualFold 'closed' (added after the existing mapped/marker checks), so github pull no longer adopts a maintainer-closed issue as a fresh open task. A mapped issue still short-circuits at the earlier mapped[is.Number] check regardless of state, so re-pull of an already-tracked closed issue is unaffected. New unit test TestShouldImportSkipsClosedUnmapped in ghmirror_test.go (alongside TestShouldImportSkipLogic) asserts a closed unmapped issue #5 is skipped and an open unmapped issue #6 still imports. go build ./... clean; go test ./internal/... all green incl. internal/features/ghmirror (2.8s). Only call site of shouldImport is the pull loop at ghmirror.go:450. Owner: verify and close via dacli task check/done + dacli merge --task 104.
