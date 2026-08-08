---
id: f-g1-g2-complete-on-branch-dacli-055-commit-8c881c3-status-labels-decisions-mirror
kind: note
note_kind: finding
created: 2026-07-22T17:10:02Z
created_by: a-wp88a6y71z
about: [[055]]
severity: major
---
# G1+G2 complete on branch dacli/055, commit 8c881c3 — status labels + decisions mirror
Commit 8c881c3 by a-wp88a6y71z, staged ONLY internal/features/ghmirror/{ghmirror.go,ghmirror_test.go} (task file stays in shared .dacli main checkout). Both pieces extend cmdPush in internal/features/ghmirror/ghmirror.go, reusing existing patterns. G1 residual: applyStatusLabel(w,num,t.Status) (ghmirror.go, called in cmdPush task loop) gives each mirrored issue EXACTLY ONE status:<folder> label (status:open|active|blocked|done via model.AllStatuses) — ensureLabel(name,--force) creates it best-effort, --add-label is idempotent, and the other three status labels are stripped so a re-push/moved task never stacks duplicates. G2: mirrorDecisions(w,project,repo,out) runs inside the SAME cmdPush after the task loop (same disclosure gate, only on explicit 'dacli github push', never on ship). Each decision note -> an issue labeled 'decision', body = choice(Chose)+Rejected+Because (the WHY) + backlink note id, keyed by decisionMarker '<!-- dacli-decision:<id> ws:<wsID> -->' (distinct prefix from task marker so mirrors never cross-adopt). REUSES searchByMarker (strong-consistency list-endpoint substring match) before create and writes 'github: issue/repo' back onto the note frontmatter via mdstore.WriteFile — mappedIssueDoc reads it so a re-push skips create. Chose labeled issues over GraphQL Discussions (decision note recorded; acceptance allows either). Tests unit-cover label dedup, marker keying (incl. no task/decision marker collision), decision mapping idempotency + WHY-in-body, empty-dir case — NO live gh required. go build ./... clean; go test -exec 'env -u DACLI_AGENT' ./internal/... all green; gofmt+vet clean. Owner: verify and close via dacli task check/done + dacli merge --task 055.
