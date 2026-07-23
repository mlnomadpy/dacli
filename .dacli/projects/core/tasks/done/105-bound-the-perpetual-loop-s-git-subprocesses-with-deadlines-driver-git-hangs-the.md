---
id: t-01KY7FYBSDJQHXAYH4APZBNCGM
kind: task
created: 2026-07-23T12:41:56Z
created_by: a-g3ya9r93e3
owner: a-root
priority: should
---
# Bound the perpetual loop's git subprocesses with deadlines (driver.git hangs the whole loop on a wedged fetch)
## So that
a wedged network or credential prompt during the always-on loop's per-cycle 'git fetch origin' can never freeze the loop indefinitely
## Acceptance
- [x] internal/features/orchestration/orchestration.go driver.git() no longer uses a bare exec.Command; every git child runs under a deadline (route through internal/gitx, or wrap in exec.CommandContext) — local ops get the short leash, the network fetch in trunkMarker() gets the network leash, matching gitx.go's localTimeout/networkTimeout
- [x] trunkMarker()'s 'git fetch -q origin <b>' is bounded so a hung fetch times out and the loop degrades to the local trunk count (its existing best-effort fallback) instead of blocking
- [x] a test proves driver.git aborts within the deadline when git hangs (e.g. inject a fake git that sleeps, assert bounded return)
- [x] go build ./... clean and go test ./internal/... green (run with DACLI_AGENT cleared)
## Log
- 2026-07-23T13:50:32Z claimed by a-k51f2ddh5e
- 2026-07-23T13:55:36Z adopted by a-root (owner a-g3ya9r93e3 orphaned)
- 2026-07-23T13:55:36Z accepted by a-root
- 2026-07-23T13:55:36Z completed by a-root
