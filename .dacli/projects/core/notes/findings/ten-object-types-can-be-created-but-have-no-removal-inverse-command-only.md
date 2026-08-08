---
id: f-ten-object-types-can-be-created-but-have-no-removal-inverse-command-only
kind: note
note_kind: finding
created: 2026-08-04T18:18:12Z
created_by: a-go-auditor-2ednq4
about: "[[t-01KZ6SAHPQ9ZB2XNTMWC3HPCV5]]"
source_event: 01KZ6SQXX25AWBMXP4KTC2X0M4
---
# ten object types can be created but have no removal/inverse command; only project/worktree/agent do
The ONLY removal or inverse commands on the whole surface are: 'project rm' (planning.go:24), 'worktree remove'/'worktree prune' (lifecycle.go:32,33), and 'agent retire' (teamops.go:29). Verified: grep -rhoE 'Path: "[^"]*(rm|remove|delete|retire|unlink|resolve|close|drop)[^"]*"' internal/features/ returns exactly those three.

CREATABLE-BUT-NOT-REMOVABLE (add with no inverse):
- queue add (queues.go:13) — no 'queue rm'; a mistyped/obsolete queue is permanent, only advanceable.
- risk add (planning.go:33) — no 'risk rm'/'risk resolve'/'risk close'; a mitigated or wrong risk stays in 'risk list' and keeps counting toward the rank-1/2 action-plan gate forever.
- shortcut add (shortcuts.go:23) / shortcut promote — no 'shortcut rm'.
- role add (teamops.go:30) — no 'role rm'; agents can retire but roles cannot, so a bad role definition is permanent (only role bump to a new version).
- template add (stagegate.go:17) — no 'template rm'.
- runtime add (execution.go:40) — no 'runtime rm'; a broken adapter cannot be removed, only re-added.
- skill add/import/fetch (skillforge.go:21,25,26) — no 'skill rm'.
- task add (planning.go:25) — no 'task rm'/'task delete'; an individual mis-filed task can only be blocked/done, never removed, even though 'project rm' can nuke a whole project. Inconsistent granularity.
- github link (ghmirror.go:42) — no 'github unlink'.
- glossary (planning.go:35) can add terms with no per-term removal path documented.

CONSEQUENCE: any workspace that accumulates mistakes (a duplicate queue, a stale risk, a typo'd role) has no in-tool remedy short of hand-editing .dacli/ files — which the tool otherwise treats as its own append-safe store. project/worktree/agent prove the inverse pattern is intended; the others are gaps.
