---
id: t-01M088WV632VPCXW0Y37P3DSCC
kind: task
created: 2026-08-17T16:29:24Z
created_by: a-root
owner: a-root
priority: must
github:
  issue: 682
  repo: mlnomadpy/dacli
estimate: "{optimistic: 5, probable: 8, pessimistic: 13}"
---
# [agent-report] github push retry can duplicate a task issue after interrupted output
## Context
Adopted from GitHub issue #682.

On 2026-08-17, `dacli github push periodica 049 050 051 052 053 054 055` created issues through #28 but returned only the plan line after the managed command ended. Retrying `dacli github push periodica 053 054 055` reported two unchanged and one created, yet created duplicate issues azettaai/perio#28 and #29 for the same task t-01M087RYC85B4PCTZ8F2BTE29N, both with the identical dacli marker. Task metadata ultimately linked #29. The mirror should reconcile all matching markers and serialize/idempotently recover interrupted pushes. Duplicate #28 was manually closed.

---
_Reported via `dacli report`._
- dacli: dev
- platform: darwin/arm64
- workspace and run transcript withheld (public upstream) — re-run with --disclose to include them

## Acceptance
- [x] An end-to-end regression interrupts after `gh issue create` succeeds but before task frontmatter is updated; the next `github push` adopts that exact issue and performs zero additional creates.
- [x] A concurrent-push regression proves two mutating pushes for the same linked repository cannot both pass the marker check and create an issue for one task.
- [x] When two pre-existing issues carry the same task marker, push creates nothing, records or reports all matching issue numbers, and chooses a deterministic canonical mapping without silently discarding the duplicate condition.
- [x] A failed, truncated, or capped marker read refuses before `issue create` and preserves the existing fail-closed behavior.
- [x] The repository lock is scoped to the linked workspace/repository, releases after success and failure, and has stale-owner recovery backed by process identity rather than elapsed time alone.
- [x] Dry-run remains read-only and accurately reports adoption, ambiguity, or refusal without acquiring a mutating lease.
- [x] Mutation evidence demonstrates the interruption and concurrent regressions fail when the pre-create reconciliation or serialization guard is removed.
- [x] Focused ghmirror tests and `go test ./...` pass.
## Log
- 2026-08-18T14:14:41Z claimed by a-maintainer-fm4hfq
- 2026-08-18T14:36:44Z accepted by a-root
- 2026-08-18T14:36:44Z verified by `env GOCACHE=/tmp/dacli-post-wave-gocache GOTMPDIR=/tmp go test ./...` (exit 0) in branch main at e82449a — proves that tree builds, not that the work is in trunk
- 2026-08-18T14:36:44Z deliverable: dacli/462-agent-report-github-push-retry-can-duplicate-a-task-issue-after-interrupted is merged into main
- 2026-08-18T14:36:44Z completed by a-root
## Verification Evidence
{"command":"env GOCACHE=/tmp/dacli-post-wave-gocache GOTMPDIR=/tmp go test ./...","exit_code":0,"duration_ms":120149,"artifact_hash":"sha256:c7943b8ac28838f67943e2578f35bc069c71957fe7769077f1148a5222d3b2b6","verifier":"a-root","branch":"main","commit_sha":"e82449acf21e69bc44b33e31a57caafd9b45c7c8"}
