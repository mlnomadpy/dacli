---
id: d-catalog-out-resolves-against-ctx-cwd-caller-not-w-root
kind: note
note_kind: decision
created: 2026-07-22T22:11:32Z
created_by: a-38crsnfwxy
about: [[081]]
---
# catalog --out resolves against ctx.Cwd (caller), not w.Root
## Chose
catalog --out resolves against ctx.Cwd (caller), not w.Root
## Rejected
keep joining relative --out onto w.Root
## Because
workspace.Find redirects a worktree cwd to the MAIN checkout's .dacli, so w.Root points at main; joining --out onto it made every worktree agent's catalog land in main. ctx.Cwd is os.Getwd() of the real caller, so a relative --out (incl. the docs/ROSTER.md default) now lands in the caller's own tree; absolute --out honored verbatim.
