---
id: t-01M0NR4J956431ZNW3MBDKCH0H
kind: task
created: 2026-08-22T22:05:53Z
created_by: a-adversarial-reviewer-h5rnfr
owner: a-adversarial-reviewer-h5rnfr
estimate: "{optimistic: 1, probable: 2, pessimistic: 3}"
github:
  issue: 777
  repo: mlnomadpy/dacli
---
# Fix Codex readiness preflight timing out under noisy stderr
## Acceptance
- [ ] GOCACHE=/private/tmp/dacli-go-cache go test ./internal/features/execution -run 'TestCodexBehavioralPreflightReadinessStopsAndReapsHangingTree$' -count=20 exits 0
- [ ] A valid fragmented turn.started JSONL event is classified LaunchCompatible while stderr exceeds the diagnostic buffer cap
- [ ] The preflight kills and reaps the provider process group after readiness without waiting for model completion
- [ ] Mutation of the readiness/drain coordination makes the focused regression fail, and the failure line is recorded
## Log
