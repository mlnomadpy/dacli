---
id: f-prcheckspass-misreads-a-red-pending-check-as-github-unreachable-and-local
kind: note
note_kind: finding
created: 2026-08-10T13:57:45Z
created_by: a-go-auditor-5yxchj
about: "[[t-01KZNYHPH9DVJJYD6P0WAVFNB5]]"
source_event: 01KZNZ9ZQ3YQY518AMJV07S4A6
---
# prChecksPass misreads a red/pending check as GitHub-unreachable and local-merges unverified code to trunk
internal/features/vcs/lifecycle.go:1393-1405 (prChecksPass) classifies a gate failure as a NETWORK failure by scanning gh's combined output.

CHAIN (verified end-to-end at HEAD):
- runGH (lifecycle.go:51-57) uses CombinedOutput(), so 'out' is gh's stdout+stderr.
- prChecksPass: out,err := runGH(root,'pr','checks',branch). err==nil => pass. Else: if isNetworkErr(out) || isNetworkErr(err.Error()) => return netErr=true (line 1398-1399).
- isNetworkErr (lifecycle.go:74-89) is a bare strings.Contains over lowercased text for tokens including 'timeout','timed out','unreachable','eof','connection reset','dial tcp'.
- gh pr checks exits NON-ZERO when a required check is failing or PENDING, printing the checks TABLE (one row per check: name, state, duration, URL) to stdout.
- Consumer prIntegrateTask (lifecycle.go:1316-1325): if netErr => prints 'GitHub unreachable', calls mergeTask() (local merge into trunk) and returns landed=true.

FAILURE SCENARIO: a repo with a CI check whose NAME contains a network token (e.g. 'e2e-timeout', 'integration-timeout', a name containing 'eof' or 'unreachable') opens a PR with that check red or still pending. gh pr checks exits 1 and prints the table containing 'timeout'/'eof'. isNetworkErr(out)=true => netErr=true => prIntegrateTask local-merges the branch into trunk and reports it INTEGRATED. Unverified/failing code lands on main and is recorded as landed, on the loop's own unattended integrate path (dacli ship --pr / integrate --pr). This defeats the exact 'an absent/failing gate is not a passed gate' guarantee dacli 216/263 claim to enforce. The 3-char token 'eof' is especially collision-prone. Same misclassification also affects the merge-failure fallbacks at lifecycle.go:1302 and 1359.

CHEAPEST FIX: do not derive netErr from the gh pr checks TABLE. A non-zero exit whose stdout is a recognizable checks listing is a gate result (pass=false), not a network condition; reserve netErr for the case where NO check listing was produced. Best: parse 'gh pr checks --json name,state' — a network error yields no JSON, a red/pending PR yields states. Verified: this is a live defect on current main, not already fixed.
