---
id: 01KY5GX01M2EP2B3TFDVFTWTYE
kind: event
event_kind: finding
created: 2026-07-22T18:20:11Z
created_by: a-drg65wknjt
about: [[t-01KY5GP5QJS16DCPAHQMTFBE5X]]
origin: agent
applied: true
---
cmdPush rewrites every task file on every push even when the mapping is unchanged

ghmirror.go:201-204: for EVERY task in the loop cmdPush unconditionally does t.Doc.Front.SetBlock(github, ...) then store.SaveTask(t) — including tasks already mapped and counted as 'unchanged' (kept). SetBlock writes the same issue/repo value, so the only effect on an already-mapped task is a redundant mdstore.WriteFile that bumps the task file's mtime on every push. mtime is a tie-breaker in ListTasks' dedup (statusRank mtime tie-break, per sibling f-duplicate-task-drift), so a push can perturb dedup ordering and touches N task files as tracked writes. Fix: only SaveTask when the github block actually changed (num differs from mappedIssue(t)).
