---
id: t-01KY7J6RSBBTPNNPXWMHQBB3EC
kind: task
created: 2026-07-23T13:21:28Z
created_by: a-0b77j7k11m
owner: a-0b77j7k11m
priority: should
---
# loop: charge idle-cycle review spawns to the token window so --window-tokens actually bounds an idling loop
## So that
the loop's rolling token budget governs its dominant steady-state cost (idle reviews) instead of leaking it
## Acceptance
- [ ] The Idle branch of loop() (orchestration.go:304-313) charges the tokens its reviewPhase spawn spends to the Governor's window (windowSpent grows across idle ticks), so Before()'s --window-tokens check can trip while idling
- [ ] dacli loop status reports a WindowTokens figure that includes idle review spend, not just runCycle spend
- [ ] The idle reviewPhase spawn also honors --max-tokens (perCycleTok) the same way build spawns do, or a decision note records why review is intentionally exempt
- [ ] A driver/governor test asserts that repeated Idle ticks accumulate window tokens and that the window guard sleeps once the budget is exceeded purely from idle-path spend
## Log
