---
id: 01KY59P3YVHHPHDF4SGSJYV831
kind: event
event_kind: finding
created: 2026-07-22T16:14:05Z
created_by: a-8b74h81fsz
about: [[t-01KY59FNENE0C7CRCSXM3WH9DD]]
origin: agent
applied: true
---
ship half-ships: accept dirties tracked .dacli tree so integrate's clean-tree guard silently no-ops, yet ship still commits+pushes the record

ship.go order is accept(1)->integrate(2)->record-commit(3). Step 1 'dacli accept --all' -> store.CloseTask -> MoveTask (store.go:633) os.Rename's TRACKED task files open/->done/ (.dacli/projects/*/tasks/** are git-tracked), leaving the tree DIRTY with tracked deletions. Step 2 integrate -> mergeTask -> gitx.Merge (gitx.go:130) refuses 'working tree is dirty' — IsClean uses --untracked-files=no which STILL reports tracked deletions. mergeTask (lifecycle.go:203) returns that plain error WITHOUT blocking the task. cmdIntegrate (lifecycle.go:271-277) treats ANY mergeTask error as a conflict, prints it, and RETURNS NIL (exit 0). In ship (ship.go:122-131) shellDacli sees exit 0 and blockedAmong finds nothing (task never blocked), so ship proceeds to commitRecord + push. Net: when accept closes any task in the same run (the designed happy path), NO branch integrates yet ship commits+pushes the .dacli record as if it succeeded — defeats the file's 'never half-ships' guarantee. Fix: commit the .dacli record BEFORE integrate, or make integrate tolerate a dirty .dacli.
