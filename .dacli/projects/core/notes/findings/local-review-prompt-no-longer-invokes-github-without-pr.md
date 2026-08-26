---
id: f-local-review-prompt-no-longer-invokes-github-without-pr
kind: note
note_kind: finding
created: 2026-08-26T13:21:03Z
created_by: a-fixer-s3nggb
about: "[[t-01M0CZANK00P2B5XY6TVJNAWCK]]"
severity: major
---
# Local review prompt no longer invokes GitHub without --pr
internal/prompts/tpl/review_workflow.md:5 now selects git show/diff of the canonical task branch in local mode; execution.go:1023 validates that branch before minting a reviewer. Mutation: changing the first template guard to {{if true}} makes TestReviewPromptUsesTaskBranchWithoutGitHubWhenPRFirstIsOff fail at prompt_ref_test.go:76.
