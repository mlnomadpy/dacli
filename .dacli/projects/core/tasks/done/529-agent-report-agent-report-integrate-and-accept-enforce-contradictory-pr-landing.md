---
id: t-01M12QX9HEPKAAS1033W6HS45D
kind: task
created: 2026-08-27T23:12:03Z
created_by: a-root
owner: a-root
github:
  issue: 841
  repo: mlnomadpy/dacli
estimate: "{optimistic: 5, probable: 8, pessimistic: 13}"
---
# [agent-report] [agent-report] integrate and accept enforce contradictory PR landing order
## Context
Adopted from GitHub issue #841.

Symptom: after PR 840 had every required GitHub check green, dacli integrate --tasks 509 --pr --merge refused because task 509 was open. Running dacli accept 509 with successful full verification then refused because the PR branch was not yet in main. The normal commands therefore form a cycle: integrate requires done, while accept requires landed. The documented playbook also requires merge plus fresh-trunk inspection before acceptance and issue closure. Suspected cause: integrate validates task status before the PR landing transaction, while accept applies its unlanded guard without recognizing a checks-passing mergeable PR or transaction context. Manual step: explicitly accept with --allow-unlanded, immediately integrate the green PR, fast-forward main, and rerun go test ./... on fresh trunk. Expected design: a single transaction such as ship --tasks should accept with deferred landing, merge only a checks-passing PR, inspect fresh trunk, record the landing verdict, and close the task; its dry-run and execution should support an explicitly selected open task without pre-rejecting it as not done. Acceptance: reproduce with a green open PR and an open fully checked task; prove the ordinary non-force path completes the transaction; prove merge failure leaves the task non-final or rolls it back; prove GitHub issue closure happens only after merged state and fresh-trunk verification; cover direct integrate/accept recovery messaging so neither command recommends an impossible next command.

---
_Reported via `dacli report`._
- dacli: dev
- platform: darwin/arm64
- workspace and run transcript withheld (public upstream) — re-run with --disclose to include them

## Acceptance
- [x] A regression fixture starts with an open, fully checked task and a green mergeable PR; the ordinary non-force ship path completes the transaction without requiring contradictory manual commands.
- [x] A PR merge/check failure leaves the task nonterminal or restores it transactionally, preserving the branch, worktree, evidence, and actionable recovery state.
- [x] The linked GitHub issue and local task close only after GitHub reports merged, fresh configured trunk contains the exact reviewed head/tree, and post-landing verification succeeds.
- [x] Direct `integrate` and `accept` refusals recommend a valid recovery command and never direct the operator into the same accept-before-merge/merge-before-accept cycle.
## Log
- 2026-08-28T13:42:00Z a-root: materialized the explicit acceptance sentence from GitHub issue #841 after adoption left the section empty; tracked general migration in #875.
- 2026-08-28T13:41:04Z claimed by a-maintainer-wg7dnx
- 2026-08-28T15:04:36Z accepted by a-root
- 2026-08-28T15:04:36Z verified by `env -u DACLI_AGENT GOCACHE=/tmp/dacli-529-postland-cache go test ./internal/features/ghmirror ./internal/features/ship ./internal/features/vcs ./internal/features/acceptance -count=1` (exit 0) in branch main at 428931a5 — proves that tree builds, not that the work is in trunk
- 2026-08-28T15:04:36Z deliverable: dacli/529-agent-report-agent-report-integrate-and-accept-enforce-contradictory-pr-landing is merged into main
- 2026-08-28T15:04:36Z completed by a-root
- 2026-08-28T15:04:38Z deliverable: dacli/529-agent-report-agent-report-integrate-and-accept-enforce-contradictory-pr-landing is merged into main
- 2026-08-28T15:15:06Z a-verifier-86vjjk: verify-verdict: no-verdict — codex-ro (a-verifier-86vjjk) on claim: Task 529 spawned with unrelated execution-only path claim — panelist reported nothing — counts as unconfirmed (event 01M14B2RJBK4A2QY9G6658NFFY)
- 2026-08-28T15:15:06Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/881 (event 01M14B3DBDYAC61QYVG4ZFTBNQ)
- 2026-08-28T15:15:06Z a-root: Landing policy override: mode=pr base=main (event 01M14ECRXJDPYA1BGGEWQC5P38)
- 2026-08-28T15:15:06Z a-root: Integrated via PR https://github.com/mlnomadpy/dacli/pull/881 at merge commit 67f94046268d2e5674fbdf1f05f7a6526346be32 into main (generation 0) (event 01M14ED127SPC4W7MGWQ4JVCC1)
## Verification Evidence
{"command":"env -u DACLI_AGENT GOCACHE=/tmp/dacli-529-accept-cache go test ./internal/features/ship ./internal/features/vcs ./internal/features/acceptance -count=1","exit_code":0,"duration_ms":13462,"artifact_hash":"sha256:e599847bd255b241e3fcac6a32898d3de75e5e5e7fe2f6c29399503c883ab625","verifier":"a-root","branch":"dacli/529-agent-report-agent-report-integrate-and-accept-enforce-contradictory-pr-landing","commit_sha":"3afd1fd74075dc28b06facfd5ed7005b85de9a2f"}
{"command":"env -u DACLI_AGENT GOCACHE=/tmp/dacli-529-accept-cache go test ./internal/features/ship ./internal/features/vcs ./internal/features/acceptance -count=1","exit_code":0,"duration_ms":11426,"artifact_hash":"sha256:a006e20e768bb4435df8a3549c0e572d34f8986f2e43122e8f582ded990b68df","verifier":"a-root","branch":"dacli/529-agent-report-agent-report-integrate-and-accept-enforce-contradictory-pr-landing","commit_sha":"9ca25c46bff989fbed8b08534d397974e7d09f4e"}
{"command":"env -u DACLI_AGENT GOCACHE=/tmp/dacli-529-final-evidence go test ./internal/features/ghmirror ./internal/features/ship ./internal/features/vcs ./internal/features/acceptance -count=1","exit_code":0,"duration_ms":13344,"artifact_hash":"sha256:724736729e1e7b918acce118feaba52fb56f0f6c011dda49e7908272ca19f553","verifier":"a-root","branch":"dacli/529-agent-report-agent-report-integrate-and-accept-enforce-contradictory-pr-landing","commit_sha":"b24d30aaa00a3bcf0bc797499a195274da963086"}
{"command":"env -u DACLI_AGENT GOCACHE=/tmp/dacli-529-postland-cache go test ./internal/features/ghmirror ./internal/features/ship ./internal/features/vcs ./internal/features/acceptance -count=1","exit_code":0,"duration_ms":13536,"artifact_hash":"sha256:8e21066d4f7dd5af87349a697e729c28c71e4c10db5428645afd4c1e9347261e","verifier":"a-root","branch":"main","commit_sha":"428931a5ef13dcf93f6affdf639b0ea7307f18b6"}
