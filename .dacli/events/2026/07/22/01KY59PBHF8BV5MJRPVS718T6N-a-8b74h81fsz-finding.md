---
id: 01KY59PBHF8BV5MJRPVS718T6N
kind: event
event_kind: finding
created: 2026-07-22T16:14:13Z
created_by: a-8b74h81fsz
about: [[t-01KY59FNENE0C7CRCSXM3WH9DD]]
origin: agent
applied: true
---
cmdIntegrate mislabels every non-conflict merge failure as a conflict and swallows it to exit 0

internal/features/vcs/lifecycle.go:271-277 — cmdIntegrate calls mergeTask and, on ANY returned error, prints '%03d-%s: conflict — %v' then 'integrated N before the conflict' and RETURNS NIL. But mergeTask returns a plain error (not the block path) for genuinely non-conflict failures: gitx.Merge propagates 'working tree is dirty' (gitx.go:130-131), 'git merge failed: <missing branch / unrelated histories / index lock / timeout>' (gitx.go:149). Those are (a) mislabeled to the user as a merge conflict, and (b) turned into exit 0, so any programmatic caller (dacli ship, ship.go:122) believes integrate succeeded. Only the TRUE conflict path in mergeTask blocks the task and returns a Refused; a non-conflict error leaves the task un-blocked AND the process exit 0. Fix: distinguish mergeTask's conflict-block return (task now blocked) from a hard error, and return the hard error (non-zero exit) instead of swallowing it.
