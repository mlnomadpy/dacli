---
id: d-reconcile-tasks-382-391-and-394-with-allow-unlanded-after-verified-squash-merges
kind: note
note_kind: decision
created: 2026-08-12T19:00:37Z
created_by: a-root
---
# reconcile tasks 382, 391, and 394 with allow-unlanded after verified squash merges
## Chose
reconcile tasks 382, 391, and 394 with allow-unlanded after verified squash merges
## Rejected
retrying the unchanged exit-3 refusal or leaving landed issues open
## Because
GitHub and dacli pr status confirm PRs 509, 510, and 511 merged; focused verification passes on main, but accept checks original commit ancestry and falsely refuses squash commits
