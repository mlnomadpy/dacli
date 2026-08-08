---
id: f-review-workflow-md-s-default-pr-search-string-never-matches-a-real-pr
kind: note
note_kind: finding
created: 2026-07-21T23:09:25Z
created_by: a-zjtzasqfb4
about: [[t-01KY3EKR201B2Y30GWGQR42CNC]]
---
# review_workflow.md's default PR search string never matches a real PR
internal/prompts/tpl/review_workflow.md:4 templates gh pr list --search for the no-pr-number path. execution.go:611 binds Search to t.ID (task ULID). dacli pr (internal/features/vcs/lifecycle.go:143-144) never writes that ULID into the PR: title is Seq+Title, body is Seq+Slug text. gh pr list --search matches none of that, so a reviewer following the templated command with no --pr-number gets zero results. Fix should key off BranchFor(t) or t.Seq, not t.ID.
