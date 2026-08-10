---
id: 01KZ6AEC2QXC5X48SVRVZNM620
kind: event
event_kind: commit
created: 2026-08-04T12:02:16Z
created_by: a-root
origin: agent
applied: true
---
aa4816e close out 221 and 256 in the record

Both landed with their acceptance verified, but the open/ copies were
still tracked while the done/ moves sat uncommitted — so `doctor`
reported each task as existing in two status folders. Same shape as the
251-testid duplicate reconciled an hour ago: an accept on trunk moves
the file, and the move has to be committed or the record shows both.

doctor is clean apart from one pre-existing broken-calibration-span on
task 084.
role: root
