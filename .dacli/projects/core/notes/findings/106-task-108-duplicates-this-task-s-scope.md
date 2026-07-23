---
id: f-106-task-108-duplicates-this-task-s-scope
kind: note
note_kind: finding
created: 2026-07-23T14:56:49Z
created_by: a-j4dcnqkbat
about: [[106]]
severity: minor
---
# 106: task 108 duplicates this task's scope
t-01KY7KRSJ9X8CV0HJP72F9EAS1 (task 108, owned by a-8p0kde6tvt) states the identical acceptance criteria as 106 (charge Idle-branch reviewPhase tokens to Governor.windowSpent, forward --max-tokens to the idle review spawn, dacli loop status reflects idle spend, regression test for repeated Idle ticks tripping SleepWindow). I implemented 106 first; the owner should close 108 as a duplicate once 106 lands rather than have a second agent redo the same fix.
