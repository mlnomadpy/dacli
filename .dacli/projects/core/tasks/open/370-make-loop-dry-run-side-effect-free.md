---
id: t-01KZV1GAHS8M803TX5M22KRMFV
kind: task
created: 2026-08-12T13:10:06Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 1, probable: 3, pessimistic: 5}"
depends_on: [372, 375]
github:
  issue: 463
  repo: mlnomadpy/dacli
---
# Make loop dry-run side-effect free
## So that
Previewing autonomous work cannot consume governor state or trip the production thrash guard
## Acceptance
- [ ] Running the same bounded loop --dry-run twice leaves the persisted cycle, streak, token window, trunk marker, and last outcome unchanged
- [ ] A dry-run cannot create or modify tasks, runs, events, worktrees, roles, runtime files, or remote state
- [ ] The preview still prints the planned build, wait, sync, ship, review, lint, retro, and doctor actions
- [ ] A regression test starts one step below the thrash threshold and proves dry-run does not cause a halt
- [ ] go test -race ./... passes
## Log
