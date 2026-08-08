---
id: f-supervise-verify-invocation-txt-omit-role-model-so-recorded-bands-can-never
kind: note
note_kind: finding
created: 2026-07-22T18:23:33Z
created_by: a-xaybvpxzth
about: [[t-01KY5GP5R2TF4MF5WS84KP18ZW]]
source_event: 01KY5H0A4K0FBK3HNQ58TKKH33
github:
  issue: 37
  repo: mlnomadpy/dacli
---
# supervise/verify invocation.txt omit role/model, so recorded bands can never match the OrDash band used by the calibrate gate/advise
cmdSpawn records the band in invocation.txt with clikit.OrDash applied (execution.go:373-374: role: %s / model: %s = '-' when empty), and the display/gate band is built the SAME way (execution.go:270: store.Band{Role: clikit.OrDash(roleName), Model: clikit.OrDash(modelName), Runtime: rt.Name}). So a spawn-completed task self-matches. But cmdSupervise's invocation.txt (execution.go:742-743) writes ONLY run/supervise_turn/task/child/runtime — no role, no model — so readInvocation yields Band{Role:"", Model:"", Runtime:rt}. verify.go:123 writes no role/model/runtime at all. Since MedianTokenRatio and printAdvisory's wall-clock fallback filter samples by exact equality 's.Band == band' (calibration.go:69, execution.go:520/542), a supervise-completed task's sample band {"","",rt} never equals the gate's OrDash band {"-","-",rt} — the two forms differ by the '-' sentinel. Net: actuals produced by dacli supervise are dead weight for by-agent-band calibration (invisible to --advise, --max-tokens, and the calibrate by-band section) even when they are NOT clobbered. Fix: write the OrDash role/model into supervise (and verify) invocation.txt exactly as cmdSpawn does, so recorded bands are in the one canonical form the gate compares against.
