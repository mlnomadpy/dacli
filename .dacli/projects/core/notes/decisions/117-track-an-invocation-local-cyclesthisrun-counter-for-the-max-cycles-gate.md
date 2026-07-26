---
id: d-117-track-an-invocation-local-cyclesthisrun-counter-for-the-max-cycles-gate
kind: note
note_kind: decision
created: 2026-07-26T22:50:56Z
created_by: a-7eanj1ps0j
about: [[117]]
---
# 117: track an invocation-local cyclesThisRun counter for the --max-cycles gate, keep the persisted cumulative cycle for reporting/resume only
## Chose
117: track an invocation-local cyclesThisRun counter for the --max-cycles gate, keep the persisted cumulative cycle for reporting/resume only
## Rejected
resetting the whole bounded counter on every fresh invocation (loses the ability to tell resume-continuation from a truly fresh run in logs/status), or adding explicit --resume/--fresh flags
## Because
the issue's own suggested direction ranks the invocation-local counter first: it fixes the reported collision (Governor.Before now gates on cyclesThisRun, which is always 0 at process start) without touching what persistence exists for (WindowSpent/windowStart for --window-tokens, zeroStreak for --no-progress-halt, cycle for status/log reporting and TestLoopRestartResumesGovernorState's resume semantics) and needs no new flags or usage-doc changes
