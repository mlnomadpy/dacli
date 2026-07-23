---
id: 01KY79EJHTFYWVCEAJZFES1X5S
kind: event
event_kind: finding
created: 2026-07-23T10:48:27Z
created_by: a-534c4gav5p
about: [[t-01KY60QM1Y7DK05WXB954YNDHJ]]
origin: agent
applied: false
---
loop --pr force-closes spawn-refused tasks, silently losing them from the backlog

In the dacli loop self-PR path, runCycle closes EVERY task in the cycle batch unconditionally, even when the implementer spawn was refused or failed. internal/features/orchestration/orchestration.go:241-254 spawns one implementer per batch task and, on error, only logs 'spawn refused/failed' (line 251-253) — it never removes the task from batch. Then orchestration.go:270-273 (the --pr LAND step) iterates the SAME batch and runs 'accept <seq> --force' for every entry. accept --force as root (acceptance.go:79-88 single-ref, and acceptOne acceptance.go:105-133) adopts the task and calls store.CheckAllAcceptance + store.CloseTask with NO gate on branch existence, an open PR, or whether acceptance criteria are actually met — it force-checks all boxes and moves the task to done. Consequence: a task whose spawn never ran (spawn refusals include PathsOverlap/--claim conflict, band-over-budget refusal, or any exec error) is marked done with zero implementation. Trunk never advances for that task, but because sibling tasks in the batch may advance trunk, the NoProgressHalt thrash-guard does not fire — so the loop keeps running and the refused task is permanently dropped from the backlog, unimplemented, and never retried. This directly contradicts the project goal 'every planned() stub implemented, honestly.' The fix: track per-task spawn success (spawn exit code / branch existence) and close ONLY the tasks whose implementer actually ran, leaving refused/failed ones open for the next cycle.
