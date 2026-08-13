---
id: f-worktree-prune-repairs-stale-registered-checkouts-shared-by-preview-and-apply
kind: note
note_kind: finding
created: 2026-08-13T23:13:20Z
created_by: a-fixer-bmmpj9
about: "[[439]]"
severity: major
---
# Worktree prune repairs stale registered checkouts shared by preview and apply
internal/cli/lifecycle_test.go:235 reproduces dry-run reporting one finished worktree followed by apply reclaiming zero when the checkout .git link is missing. internal/gitx/gitx.go:299 now prunes the registered stale entry, verifies Git no longer owns it, and removes the orphan directory. Mutation with the old RemoveWorktree body failed at lifecycle_test.go:271 with 'reclaimed 0 worktree(s)'. gofmt -l ., go vet ./..., and go test ./... passed; golangci-lint was unavailable (command not found).
