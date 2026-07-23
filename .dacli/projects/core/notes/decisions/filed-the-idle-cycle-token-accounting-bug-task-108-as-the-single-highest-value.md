---
id: d-filed-the-idle-cycle-token-accounting-bug-task-108-as-the-single-highest-value
kind: note
note_kind: decision
created: 2026-07-23T13:49:03Z
created_by: a-8p0kde6tvt
about: [[084]]
---
# Filed the idle-cycle token-accounting bug (task 108) as the single highest-value change
## Chose
Filed the idle-cycle token-accounting bug (task 108) as the single highest-value change
## Rejected
Filing the perpetual-loop git-no-deadline freeze (f-g3ya9r93e3, driver.git orchestration.go:506-511 has no context; trunkMarker:492 does a per-cycle network 'git fetch origin') as the single change instead
## Because
I independently verified BOTH findings live in code. Both are real defects in the always-on loop, but severity×probability favors the token bug: it defeats a user-configured safety guard (--window-tokens) with CERTAINTY on the loop's dominant self-feeding idle path (backlog-empty -> Before returns Idle at governor.go:134 -> loop 307-316 runs a token-spending reviewPhase then continues, never calling AfterCycle, the sole writer of windowSpent at governor.go:144), needing no trigger, causing unbounded unaccounted cost every idle cycle. The git freeze is catastrophic but CONDITIONAL (needs a hung fetch/credential prompt). The git-deadline finding remains filed for separate follow-up.
