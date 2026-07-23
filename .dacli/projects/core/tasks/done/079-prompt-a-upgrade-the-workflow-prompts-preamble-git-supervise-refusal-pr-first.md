---
id: t-01KY5X8BKEQRVSMGTGSR5K29VY
kind: task
created: 2026-07-22T21:56:06Z
created_by: a-root
owner: a-root
priority: should
estimate: {optimistic: 3, probable: 5, pessimistic: 8}
---
# PROMPT-A: upgrade the workflow prompts (preamble, git, supervise, refusal) — PR-first + newest features + better framing
## Acceptance
- [x] git_workflow.md makes PR-FIRST the default: after committing, push the branch and open a PR via dacli pr (which links Fixes #issue + posts verify verdicts), not just a local commit; --claim scope discipline is stated; keep the isolation directive
- [x] protocol_preamble.md frames the full lifecycle crisply (claim -> work -> commit -> pr -> accept/ship), references the trust-floor a finding carries, and PRESERVES the HEADLESS + WORKSPACE ISOLATION clauses verbatim
- [x] supervise_correction.md and refusal_next.md are accurate to current behavior; better framing throughout (clear, assertive, no stale command names)
- [x] committed by an agent and opened as a PR; go build + go test ./internal/... green (prompt tests must pass)
## Log
- 2026-07-22T21:56:23Z claimed by a-gjzyj2dym5
- 2026-07-22T22:03:23Z accepted by a-root
- 2026-07-22T22:03:23Z completed by a-root
- 2026-07-22T23:52:35Z a-gjzyj2dym5: PR opened: https://github.com/mlnomadpy/dacli/pull/41 (event 01KY5XJJ3RAT32GB8DKV797CA9)
