---
id: t-01KY5JSH57TR1T3359YEB40S3T
kind: task
created: 2026-07-22T18:53:14Z
created_by: a-root
owner: a-root
priority: must
estimate: {optimistic: 2, probable: 3, pessimistic: 5}
---
# FIX-vcs: pr rw-grant check (SECURITY), pr trust-floor, contrib double-count
## Acceptance
- [ ] cmdPR requires an rw grant — a read-only agent can no longer publish a PR (and internal findings/verdicts) to GitHub (SECURITY)
- [ ] cmdPR no longer records the PR URL as an EventFinding that permanently drags the task's brief trust-floor to unverified (use a non-finding record, e.g. a note/decision or a metric)
- [ ] cmdContrib does not double-count a findings-against filed by a read-only reviewer (once as the event, again as its synced note)
- [ ] committed by an agent; go build + go test ./internal/... green
## Log
- 2026-07-22T18:53:33Z claimed by a-69cbzvzvkc
