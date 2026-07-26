---
id: f-146-complete-on-branch-dacli-146-commit-defbeff-both-acceptance-criteria-met
kind: note
note_kind: finding
created: 2026-07-26T15:22:03Z
created_by: a-94k1g003b6
about: [[146]]
severity: moderate
---
# 146: complete on branch dacli/146-..., commit defbeff — both acceptance criteria met
Committed defbeff on branch dacli/146-dacli-pr-render-trust-grade-verdict-tally-loudly-into-the-pr-research-shortlist. (1) dacli pr --with-verdicts now renders a loud trust-grade summary + per-finding verdict tally as a first-class section in BOTH the PR body (prBody, internal/features/vcs/lifecycle.go: new trustGradeSection, placed right after the intro line, ahead of Acceptance/Findings) and the PR review (verdictReview, prepended ahead of the existing per-seat verdict list) — plain dacli pr with no --with-verdicts is unaffected. It aggregates the task's finding-note trust: frontmatter (confirmed/unverified/refuted counts + a trust floor using the same refuted<unverified<confirmed ordering as internal/brief's D3 trust-floor — exported as brief.TrustLabel/TrustRank/RankTrust and reused, not duplicated) and joins each finding's title to its recorded verify-verdict: comment events (parseVerdictLine/verdictTally) for a per-finding 'N confirmed, M refuted' tally. No new data collection — reads only what dacli verify + store.GradeFinding already write. (2) Covered by 2 new tests: TestPRBodyWithVerdictsRendersTrustGradeAndTally and TestVerdictReviewLeadsWithTrustGrade (internal/features/vcs/lifecycle_test.go), plus all 4 pre-existing pr/verdict tests still pass unmodified (only their prBody call sites gained an explicit withVerdicts=false arg). go build ./... clean; go test ./... all green (incl. internal/cli's TestFeatureSlicesAreIsolated — internal/brief is not a feature slice, so importing it from vcs is architecturally legal, confirmed by a green build). Owner: verify and close via task check/done + dacli merge --task 146.
