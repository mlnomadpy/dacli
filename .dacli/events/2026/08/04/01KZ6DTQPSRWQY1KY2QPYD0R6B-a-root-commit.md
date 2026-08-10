---
id: 01KZ6DTQPSRWQY1KY2QPYD0R6B
kind: event
event_kind: commit
created: 2026-08-04T13:01:27Z
created_by: a-root
origin: agent
applied: true
---
f5e7752 file 263: CI silently failed to trigger on 3 of this session's PRs

PRs 297 (242), 322 (224) and 327 (261) each reported 'no checks
reported on the branch' — no workflow run existed for the head SHA at
all, not a failing one. The workflow triggers on a bare `pull_request:`
with no filters, so every PR should get one.

What the three share is the order of operations: the branch was pushed
to origin first and `gh pr create` opened the PR seconds later. Most
PRs opened exactly that way DID get checks, so it is not deterministic —
it reads as a race where the pull_request event lands with no new commit
to run against.

The recovery is the same every time: merge origin/main into the branch
and push. A new head SHA produces a synchronize event and checks run.
`gh pr update-branch` does this too but refuses on conflict, which was
the case for all three, so each needed a hand-resolved merge first.

The dangerous part is the silence, not the missing run. `gh pr checks`
says 'no checks reported', mergeStateStatus says UNKNOWN, and nothing
flags it. Task 216 already made dacli treat 'no checks' as NOT passing,
so integrate correctly leaves the PR open — but never says why it will
sit there forever.
role: root
