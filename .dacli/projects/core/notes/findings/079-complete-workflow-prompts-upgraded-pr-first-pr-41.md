---
id: f-079-complete-workflow-prompts-upgraded-pr-first-pr-41
kind: note
note_kind: finding
created: 2026-07-22T22:01:53Z
created_by: a-gjzyj2dym5
about: [[079]]
severity: moderate
---
# 079 complete: workflow prompts upgraded (PR-first), PR #41
Commit 6f75caa (a-gjzyj2dym5). Staged ONLY the 4 claimed templates via git add + dacli commit --no-add: internal/prompts/tpl/{git_workflow.md,protocol_preamble.md,supervise_correction.md,refusal_next.md}. All 4 acceptance criteria met: (1) git_workflow.md makes PR-FIRST the default close-out — the .PR branch leads with 'push --task then pr --task --with-verdicts' (pr body carries acceptance+findings+Fixes #issue, --with-verdicts posts verify verdicts as a PR review), not a local commit; a new 'Stay inside your claim' bullet states --claim scope discipline; the isolation worktree directive is preserved. (2) protocol_preamble.md now frames the full lifecycle 'claim -> work -> commit -> pr -> accept/ship' up top, and the findings bullet references the trust-floor (unverified until a verify panel grades it, refuted<unverified<confirmed); the HEADLESS and WORKSPACE ISOLATION clauses are preserved VERBATIM. (3) supervise_correction.md and refusal_next.md reframed sharper/assertive with no stale command names (kept literal 'supervisor: turn N' and 'Do NOT push or open a pull request' that cli tests assert). (4) committed by an agent + opened as PR #41; go build ./... clean and go test ./internal/... green (0 FAIL, incl. internal/prompts, internal/cli TestWorkflowPromptsReachChildren + TestSuperviseLoopConverges). Owner: verify and check boxes via dacli accept/task check + close.
