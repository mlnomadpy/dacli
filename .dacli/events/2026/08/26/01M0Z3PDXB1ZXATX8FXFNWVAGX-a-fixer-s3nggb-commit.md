---
id: 01M0Z3PDXB1ZXATX8FXFNWVAGX
kind: event
schema_version: 1
event_kind: commit
created: 2026-08-26T13:21:03Z
created_by: a-fixer-s3nggb
about: "[[t-01M0CZANK00P2B5XY6TVJNAWCK]]"
origin: agent
applied: true
checksum: sha256:027a4e7cf17944afbd7e3f4948d5832246c9870b08d2602dedc2ffe307282ec5
---
a18e8e1 t-01M0CZANK00P2B5XY6TVJNAWCK: review local task branches without GitHub PRs

When PR-first is off, reviewers previously received an unconditional gh pr list instruction and detached without an actionable result if no PR existed. Select the canonical task branch and local git diff instead, while retaining the GitHub workflow under --pr. Mutation proof: changing the local/PR template guard to unconditional PR mode made TestReviewPromptUsesTaskBranchWithoutGitHubWhenPRFirstIsOff fail at prompt_ref_test.go:76.
role: fixer
