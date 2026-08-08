---
id: f-g3-dacli-pr-enriches-body-acceptance-findings-fixes-issue-and-mirrors-verify
kind: note
note_kind: finding
created: 2026-07-22T17:26:37Z
created_by: a-hwe0pzgt19
about: [[056]]
severity: moderate
---
# G3: dacli pr enriches body (acceptance+findings+Fixes #issue) and mirrors verify verdicts as PR review comments
internal/features/vcs/lifecycle.go: prBody() assembles acceptance (taskAcceptance) + task findings (taskFindings, filtered by about) + Fixes #N from the task's own github: frontmatter block (taskFixesLine, parsed directly — no ghmirror import); skips cleanly when unlinked. New --with-verdicts flag on 'dacli pr' posts verify verdicts as a 'gh pr review --comment' (exec.CommandContext 120s timeout, operator-triggered only). internal/features/execution/verify.go: each panel seat now records its verdict as a queryable EventComment about the task via the exported VerdictRecord()/VerdictMarker convention ('verify-verdict: ...'). verdictReview() in vcs reads that convention back (slices don't import each other). lifecycle_test.go unit-tests body assembly + verdict rendering on fixtures, no live gh. go build + go test ./internal/... green.
