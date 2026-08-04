---
id: 01KZ6B1FF4C1VWH44DMRZ2FJ3R
kind: event
event_kind: commit
created: 2026-08-04T12:12:42Z
created_by: a-root
origin: agent
applied: false
---
1fc73c5 file 261, and correct the ship guidance I wrote one command too early

I added 'prefer ship over hand-running accept then integrate' to the
skill, then ran `ship --dry-run` here and watched it build a 250-ref
integrate command covering every task ever closed on this project — not
the four I had just landed.

Most of those branches are gone and skip harmlessly, but some still
exist locally, and re-merging an old branch is not a no-op you want to
discover afterwards. So ship is right on a young project and gets more
dangerous the longer one runs, which is the opposite of what a
tie-off command should do.

261 is the fix: ship should integrate the tasks THIS run closed, or an
explicit window, not the full done set.

The skill now says both halves — ship closes the record-commit gap that
keeps biting, and on a project with history you read `ship --dry-run`
first and close the wave by hand when the integrate line looks like
that. Advice that only holds for the first month is worse than no
advice.
role: root
