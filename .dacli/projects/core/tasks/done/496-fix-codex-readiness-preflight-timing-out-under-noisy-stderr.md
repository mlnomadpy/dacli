---
id: t-01M0NR4J956431ZNW3MBDKCH0H
kind: task
created: 2026-08-22T22:05:53Z
created_by: a-adversarial-reviewer-h5rnfr
owner: a-root
estimate: "{optimistic: 1, probable: 2, pessimistic: 3}"
github:
  issue: 777
  repo: mlnomadpy/dacli
---
# Fix Codex readiness preflight timing out under noisy stderr
## Acceptance
- [x] GOCACHE=/private/tmp/dacli-go-cache go test ./internal/features/execution -run 'TestCodexBehavioralPreflightReadinessStopsAndReapsHangingTree$' -count=20 exits 0
- [x] A valid fragmented turn.started JSONL event is classified LaunchCompatible while stderr exceeds the diagnostic buffer cap
- [x] The preflight kills and reaps the provider process group after readiness without waiting for model completion
- [x] Mutation of the readiness/drain coordination makes the focused regression fail, and the failure line is recorded
## Log
- 2026-08-26T13:12:41Z takeover by a-root from a-adversarial-reviewer-h5rnfr (recovery: task takeover --force; reason: owner reviewer finished after filing issue 777; no live process or transcript-active run remains)
- 2026-08-26T13:21:21Z claimed by a-root (event 01M0Z199MA7VKDS40P27FZGWX4)
- 2026-08-26T13:58:06Z claimed by a-fixer-qhwq3c
- 2026-08-26T14:15:11Z completed by a-root
- 2026-08-26T14:29:48Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/788 (event 01M0Z6CCPXW2A0QBCSK74H5TFB)
## Verification Evidence
{"command":"GOCACHE=/private/tmp/dacli-go-cache go test ./internal/features/execution -run TestCodexBehavioralPreflightReadinessStopsAndReapsHangingTree -count=20","exit_code":0,"duration_ms":5978,"artifact_hash":"sha256:812a7059317aac61826ae74e4db0f49b08a6e46033df323436f5aaba437fc4d4","verifier":"a-root","branch":"dacli/496-fix-codex-readiness-preflight-timing-out-under-noisy-stderr","commit_sha":"b841352b7330ae4035430a6aa5e626e16d5359d7"}
