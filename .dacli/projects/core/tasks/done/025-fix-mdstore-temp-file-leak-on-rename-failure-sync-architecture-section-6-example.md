---
id: t-01KY4YCXPCE6RYKKJQY2HXZY6C
kind: task
created: 2026-07-22T12:56:50Z
created_by: a-root
owner: a-root
priority: could
estimate: {optimistic: 1, probable: 1, pessimistic: 2}
---
# Fix mdstore temp-file leak on rename failure; sync ARCHITECTURE section 6 example
## Context
Real audit findings. Anchors:
- `internal/mdstore/mdstore.go:453-471` `WriteFile` writes a temp file then `return os.Rename(name, path)`. It removes the temp on the write-error (:464) and close-error (:468) paths, but NOT when `os.Rename` fails — so a failed rename leaks `.dacli-tmp-*` files into object dirs. Wrap the rename: on error, `os.Remove(name)` before returning it.
- `docs/ARCHITECTURE.md` §6 shows a canonical brief example, but the assembler (`internal/brief/brief.go`) also emits two sections the example omits: "Lessons from other projects" (inserted between Glossary and What-siblings-found) and "Recent activity" (between What-siblings-found and Shortcuts). Add both to the §6 example in their correct positions so the doc matches what a brief actually contains. Doc-only — do not change brief.go.

## Scope (STRICT) — touch ONLY:
- `internal/mdstore/mdstore.go`
- `docs/ARCHITECTURE.md`

## Staging discipline
Do NOT `git add -A`. `git add` ONLY the two files above plus this task's file. `go build ./...` + `go test ./internal/...` green. Paste summary as `dacli note add finding`, `dacli commit`, then `dacli task check`.

## Acceptance
- [x] mdstore.WriteFile removes its temp file when os.Rename fails (currently leaks .dacli-tmp-* into object dirs)
- [x] docs/ARCHITECTURE.md section 6 brief example includes the Lessons-from-other-projects and Recent-activity sections the assembler actually emits
- [x] committed on branch by an agent; go build + go test ./internal/... green
## Log
- 2026-07-22T13:03:24Z completed by a-root
