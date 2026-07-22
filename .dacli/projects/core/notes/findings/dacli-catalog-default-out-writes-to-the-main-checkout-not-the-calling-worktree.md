---
id: f-dacli-catalog-default-out-writes-to-the-main-checkout-not-the-calling-worktree
kind: note
note_kind: finding
created: 2026-07-22T20:53:59Z
created_by: a-0bsqxp2kpx
about: [[069]]
severity: minor
---
# dacli catalog default --out writes to the MAIN checkout, not the calling worktree (worktree .dacli shadowing)
Because workspace.Find redirects a linked-worktree cwd to the main worktree's .dacli and sets w.Root to the MAIN root, a relative --out (default docs/ROSTER.md) is joined against the main root — so 'go run ./cmd/dacli catalog' from a worktree wrote docs/ROSTER.md into the MAIN checkout, not the worktree branch (a stray untracked file now sits in the main tree; harmless generated sample, owner can rm it). This is the same worktree-shadows-shared-dacli-workspace behavior noted in memory, not a catalog bug: an absolute --out (used for the committed sample) lands correctly. If desired, catalog could resolve --out against the git worktree cwd rather than w.Root, but that would diverge from how every other command resolves paths.
