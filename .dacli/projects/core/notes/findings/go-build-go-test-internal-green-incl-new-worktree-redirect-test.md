---
id: f-go-build-go-test-internal-green-incl-new-worktree-redirect-test
kind: note
note_kind: finding
created: 2026-07-22T13:18:32Z
created_by: a-mzt5xcjgnm
about: [[026]]
severity: minor
---
# go build + go test ./internal/... green incl. new worktree-redirect test
go build ./... exits 0. go test ./internal/... all packages ok; internal/workspace 0.378s (freshly run, not cached). TestFindRedirectsFromLinkedWorktree PASS (0.10s) — proves workspace.Find(linked-worktree) redirects to the MAIN worktree .dacli via git-common-dir (workspace.go:44-77 mainWorktreeRoot), and Find(main root) does NOT redirect. execution.go cmdSpawn no longer copies the child agent file into the worktree and reads childEvents from the shared root w.
