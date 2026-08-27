---
id: t-01M12K8SEVWQQJXS5MBPMTJWNR
kind: task
created: 2026-08-27T21:50:56Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
---
# Fix reopened task integration ignoring the new PR generation
## Acceptance
- [x] PR resolution prefers an open PR whose head matches the current canonical task branch and generation over a historical merged PR.
- [x] dacli pr status and dacli integrate select the follow-up PR for a reopened task and do not report already landed from the previous generation.
- [x] A regression fixture covers a completed task reopened to generation 1 with both a historical merged PR and a current open PR.
- [x] Existing single-generation and multi-slice PR resolution tests continue to pass, along with go test ./...
## Log
- 2026-08-27T21:53:04Z claimed by a-fixer-zpvnda
- 2026-08-27T22:11:05Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/835 (event 01M12MC4ANV1MYWK0DYX203454)
- 2026-08-27T22:20:02Z accepted by a-root (applied 1 proposal(s))
- 2026-08-27T22:20:02Z verified by `env GOCACHE=/tmp/dacli-accept-go-cache go test ./...` (exit 0) in branch main at 483b32d4 — proves that tree builds, not that the work is in trunk
- 2026-08-27T22:20:02Z deliverable: dacli/525-fix-reopened-task-integration-ignoring-the-new-pr-generation is merged into main
- 2026-08-27T22:20:02Z completed by a-root
## Verification Evidence
{"command":"env GOCACHE=/tmp/dacli-accept-go-cache go test ./...","exit_code":0,"duration_ms":78912,"artifact_hash":"sha256:9aa45786db3769e888d25b2ff48e32ea1698484fe09ea9fd6698664300699af7","verifier":"a-root","branch":"main","commit_sha":"483b32d4db733ff4dace381474773ff2bb2babb0"}
