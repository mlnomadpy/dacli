---
id: 01KZ6SR9HXKSCD16T8XX9AAXNB
kind: event
event_kind: finding
created: 2026-08-04T16:29:50Z
created_by: a-go-auditor-2ednq4
about: "[[t-01KZ6SAHPQ9ZB2XNTMWC3HPCV5]]"
origin: agent
applied: true
---
every github remote-mutating command lacks --dry-run, while local integrate/ship/loop/worktree-prune offer one

MUTATING COMMANDS THAT OFFER --dry-run (the local-side ones): merge/integrate (lifecycle.go:186), ship (ship.go:82), loop (orchestration.go:133), worktree prune (lifecycle, via --dry-run), skill compile (skillforge.go:333), run (shortcuts.go:134,207).

MUTATING COMMANDS WITH NO --dry-run — notably the remote-mirror family, which is the highest-blast-radius surface: grep -c 'dry-run' internal/features/ghmirror/*.go is 0 across ALL of them:
- github push (ghmirror.go:43) — mirrors every local task to a real GitHub issue + posts finding comments; no preview of what it will create/comment.
- github sync (ghmirror.go:44) — pull then push, same.
- github pull (ghmirror.go:45) — adopts remote issues as local tasks.
- github project (ghmirror.go:46) — writes a Project v2 board.
- github release (ghmirror.go:47) — cuts a tagged release.
Also no dry-run: accept --all (acceptance.go:64, closes every proposed task at once), spawn (execution.go, launches a real child process), note add, commit, push.

CONSEQUENCE: a mis-scoped 'github push' or 'github sync' writes to a public/shared remote with no way to preview the diff first — and the project's own risk register flags 'Public-repo mirror leaks internal findings' as rank 2. A --dry-run on the mirror commands is the cheapest guard against that risk. This is the audit's 'mutates but offers no --dry-run' list; the github family is the part worth acting on.
