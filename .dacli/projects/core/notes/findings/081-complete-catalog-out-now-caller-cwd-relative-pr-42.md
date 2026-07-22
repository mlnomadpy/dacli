---
id: f-081-complete-catalog-out-now-caller-cwd-relative-pr-42
kind: note
note_kind: finding
created: 2026-07-22T22:12:00Z
created_by: a-38crsnfwxy
about: [[081]]
severity: moderate
---
# 081 complete: catalog --out now caller-cwd relative; PR #42
Commit bbe0a49 on branch dacli/081-... (PR #42). Fix: internal/features/catalog/catalog.go cmdCatalog previously joined a relative --out onto w.Root (catalog.go:74-76), but workspace.Find redirects a worktree cwd to the MAIN checkout, so w.Root = main and every worktree agent's catalog landed in main. New pure helper resolveOut(cwd, out) (catalog.go:41-56) joins a relative --out (incl. the docs/ROSTER.md default) onto ctx.Cwd (os.Getwd() of the real caller); absolute --out honored verbatim. The 'wrote <rel>' line now reports relative to ctx.Cwd. Both acceptance criteria met: (1) default resolves to cwd/docs/ROSTER.md; (2) committed by agent + PR opened. go build ./... clean; go test ./internal/... all green incl. new TestResolveOut. Box-checking refused for non-owner (only a-root) — owner should verify and close via dacli task check/done + merge --task 081.
