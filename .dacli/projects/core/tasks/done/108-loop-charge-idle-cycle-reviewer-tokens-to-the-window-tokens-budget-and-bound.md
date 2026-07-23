---
id: t-01KY7KRSJ9X8CV0HJP72F9EAS1
kind: task
created: 2026-07-23T13:48:48Z
created_by: a-8p0kde6tvt
owner: a-root
priority: should
---
# loop: charge idle-cycle reviewer tokens to the --window-tokens budget (and bound them with --max-tokens)
## So that
the loop's steady-state token guard actually limits spend instead of being silently defeated on the dominant idle path
## Acceptance
- [x] Idle-cycle reviewer token spend is summed into the governor window (windowSpent) so the --window-tokens guard can trip on the idle path — today the Idle branch (internal/features/orchestration/orchestration.go:307-316) calls reviewPhase() then continues, never reaching AfterCycle (governor.go:142-144), the only writer of windowSpent
- [x] When --max-tokens/perCycleTok is set, the reviewPhase spawn (orchestration.go:546) forwards --max-tokens, mirroring the BUILD spawn (orchestration.go:386-388), so idle review runs are bounded per-run not just uncharged
- [x] dacli loop status (saveState, orchestration.go:263) reports non-zero window spend after idle cycles that ran a reviewer
- [x] A regression test asserts windowSpent grows across idle cycles that spawn a reviewer and that the --window-tokens guard eventually trips (SleepWindow) on a purely-idle loop
- [x] The self-feeding idle-review behavior is unchanged (a review still runs each idle cycle) — only its token accounting and per-run bounding change
## Log
- 2026-07-23T14:51:52Z adopted by a-root (owner a-8p0kde6tvt orphaned)
- 2026-07-23T14:51:52Z accepted by a-root
- 2026-07-23T14:51:52Z completed by a-root
