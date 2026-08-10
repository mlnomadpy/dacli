---
id: 01KZ65WB0YNHJXZ89DD5NJRDGH
kind: event
event_kind: commit
created: 2026-08-04T10:42:31Z
created_by: a-root
origin: agent
applied: true
---
5032b75 size the open backlog so CPM and next --parallel actually run

Every open task was unestimated, so `dacli critical-path` refused
outright and `dacli next --parallel` silently degraded to
MoSCoW-then-sequence — which is to say the two commands that justify
three-point estimation had never once run on this project's own
backlog. Seventeen tasks now carry o/m/p.

Also records this wave's swarm: six agent identities and their commit
events.

Two tasks (244, 247) are owned by agents and correctly refused sizing
by root without --force; they stay unsized. That single gap is enough
to keep critical-path refusing, which is itself worth knowing — the
command is all-or-nothing over the open set.

Estimates for 205, 210, 215, 243 and 248 are deliberately NOT here:
those tasks are in flight on their own branches and re-sizing them from
main would hand every one of those PRs a conflict in its own task file.
role: root
