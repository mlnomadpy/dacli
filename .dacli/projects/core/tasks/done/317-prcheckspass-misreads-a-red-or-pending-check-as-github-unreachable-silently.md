---
id: t-01KZNZB1AXYPH977FTQ7ZA0B02
kind: task
created: 2026-08-10T13:56:04Z
created_by: a-go-auditor-5yxchj
owner: a-root
priority: must
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
---
# prChecksPass misreads a red or pending check as GitHub-unreachable, silently merging unverified code to trunk and reporting it landed
## So that
the loop's unattended integrate gate cannot be tricked into landing unverified code and recording it as merged
## Acceptance
- [x] prChecksPass (internal/features/vcs/lifecycle.go:1393) no longer sets netErr=true from the gh pr checks TABLE: when gh pr checks exits non-zero because a check is failing or pending, it returns pass=false, netErr=false (a gate result), reserving netErr for a genuine GitHub-unreachable condition where no check listing was produced (e.g. via gh pr checks --json name,state, or by only classifying err/stderr when stdout carries no check rows)
- [x] A regression test drives prChecksPass with a gh stub that exits non-zero and prints a checks table containing a check whose name carries a network token (e.g. e2e-timeout, or a name containing eof/unreachable); it asserts pass=false AND netErr=false
- [x] A test confirms prIntegrateTask (lifecycle.go:1316) does NOT call mergeTask and returns landed=false for a red/pending PR whose checks-table output contains a network token — the branch is left open, not local-merged to trunk
- [x] The same misclassification is closed (or shown not to apply) for the merge-failure fallbacks at lifecycle.go:1302 and 1359, which also scan isNetworkErr(out)
## Log
- 2026-08-10T14:30:09Z adopted by a-root (owner a-go-auditor-5yxchj orphaned)
- 2026-08-10T14:30:09Z accepted by a-root (applied 1 proposal(s))
- 2026-08-10T14:30:09Z verified by `go test ./internal/features/vcs/...` (exit 0)
- 2026-08-10T14:30:09Z deliverable: no dacli/317-prcheckspass-misreads-a-red-or-pending-check-as-github-unreachable-silently branch — nothing to check against trunk
- 2026-08-10T14:30:09Z completed by a-root
