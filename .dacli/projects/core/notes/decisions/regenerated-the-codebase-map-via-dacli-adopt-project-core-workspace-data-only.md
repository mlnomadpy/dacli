---
id: d-regenerated-the-codebase-map-via-dacli-adopt-project-core-workspace-data-only
kind: note
note_kind: decision
created: 2026-07-23T13:16:30Z
created_by: a-rd1mwxdxpf
about: [[101]]
---
# Regenerated the codebase map via 'dacli adopt --project core' (workspace data only), no source commit/branch/PR
## Chose
Regenerated the codebase map via 'dacli adopt --project core' (workspace data only), no source commit/branch/PR
## Rejected
adding a no-op source change just to have something to commit+PR through the normal task branch flow
## Because
acceptance criteria 101 are entirely about the STORED DATA state (project.md's Open markers block, and fresh briefs) — the scanner itself was already fixed in task 087, so no code changes are needed here. dacli commit runs git in ctx.Cwd (vcs.go:85, the agent's own worktree), never in w.Root, while workspace commands (adopt, note, task) always redirect to the shared main-checkout .dacli (task 026's fix) — so a data-only regeneration physically cannot be captured by a task-branch commit; it lands as an uncommitted change on the main checkout, the same place every 'ship: record workspace...' commit comes from. Verification (grep + brief inspection) is recorded as a finding instead.
