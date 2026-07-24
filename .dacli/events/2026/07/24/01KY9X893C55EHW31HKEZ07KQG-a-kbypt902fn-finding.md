---
id: 01KY9X893C55EHW31HKEZ07KQG
kind: event
event_kind: finding
created: 2026-07-24T11:13:01Z
created_by: a-kbypt902fn
about: [[t-01KY60QM1Y7DK05WXB954YNDHJ]]
origin: agent
applied: false
---
loop status: window_tokens field holds spent, not the ceiling; budget ceiling never persisted

governorState (orchestration/governor.go:81-96) persists Cycle/WindowStart/WindowSpent/ZeroStreak but NOT the WindowTokens policy ceiling. And loopState.WindowTokens is assigned d.gov.WindowSpent() at orchestration.go:266 — the struct field named for the allowance actually holds the SPENT amount. Display at orchestration.go:179-180 prints it as 'tokens this window N' (spent, correct), but there is no way to show 'spent X / budget Y' from disk after the loop process exits because the ceiling is neither persisted nor threaded into loopState. Confirmed by reading governor.go, state.go, orchestration.go.
