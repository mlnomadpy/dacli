---
id: f-burn-rate-counts-non-completing-implementer-runs-the-ceiling-excludes-re
kind: note
note_kind: finding
created: 2026-08-04T00:37:40Z
created_by: a-ham9kmbg3b
about: "[[t-01KY60QM1Y7DK05WXB954YNDHJ]]"
source_event: 01KYGAVA926NR8HJ0KV87SGRWE
---
# burn Rate counts non-completing implementer runs the Ceiling excludes, re-opening the 149/153 false-negative on a new axis
burnSeries (internal/features/dashboard/burn.go:138-186) computes the day PerRun over EVERY implementer run with usage.txt output>0 (role not verify/reviewer/ro), then Rate = latest day PerRun (burn.go:107,181: tokens/Runs). But the Ceiling is store.LoadCalibration (internal/store/calibration.go:149-178): ONE sample per DONE task, tokens = that task's completing run only (runRecords keeps the last usage-bearing run per task; only tasks in ListTasks StatusDone with a valid estimate+logSpan). The populations differ on the COMPLETION axis, not the role axis 149/153 fixed: (1) a failed/refused implementer run that still burned tokens, (2) every implementer run of a task not yet done, and (3) earlier retry runs of a task that took multiple runs, are all counted in Rate but NEVER in Ceiling. Effect: retry/failed/in-flight runs inflate the Rate denominator and add sub-ceiling samples, biasing Ratio below 1.0 and re-opening the exact false-negative (silent overspend, Alert=false when it should yell) task 149 was filed to close. The burn.go:128-137 comment asserts Rate is 'counted over the SAME population the Ceiling is built from' -- literally false: 149 filtered by ROLE (verify/review), 153 by ro-grant; neither filters by task-status/completion. Unit-testable: a day with one 50k completing run of a done task plus two 5k failed implementer runs of an abandoned task yields Rate=(50k+5k+5k)/3=20k vs Ceiling=50k, Ratio=0.4, Alert=false even though the real completing-run cost equals the ceiling.
