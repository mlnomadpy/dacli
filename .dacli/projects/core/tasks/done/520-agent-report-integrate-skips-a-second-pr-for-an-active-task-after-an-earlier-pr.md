---
id: t-01M11HZW8X678270CFWDBK7YTN
kind: task
created: 2026-08-27T12:09:21Z
created_by: a-root
owner: a-root
github:
  issue: 813
  repo: mlnomadpy/dacli
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
---
# [agent-report] integrate skips a second PR for an active task after an earlier PR landed
## Context
Adopted from GitHub issue #813.

An active task intentionally used multiple bounded PR slices. PR #89 landed first while the parent task remained open. A fresh worktree/branch then produced fully green PR #90 for the same active task. dacli integrate --tasks <task> --pr --into dev --merge --force reported already landed using PR #89, ignored open PR #90, and cleaned the local worktree. The workaround was gh pr merge 90 --merge. Expected: integration status should resolve the current branch/open PR, not treat any historical landed PR for an active task as proof that all later task work landed.

---
_Reported via `dacli report`._
- dacli: dev
- platform: darwin/arm64
- workspace and run transcript withheld (public upstream) — re-run with --disclose to include them

## Acceptance
- [x] A regression fixture creates a task with one historical merged PR and a newer open PR on the current canonical task head; integration selects the newer PR and does not report the task already landed.
- [x] A historical merged PR is used only when its head/tree is the task's current delivery generation; closed-unmerged and superseded generations cannot prove landing.
- [x] Worktree cleanup is not performed while a current unlanded PR generation exists.
- [x] The regression fails when integration returns on the first historical merged PR, and focused plus repository-wide Go quality gates pass.
## Log
- 2026-08-28T12:47:40Z claimed by a-fixer-fzgpye
- 2026-08-28T13:01:19Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/865 (event 01M146YPAB383N8D7CPNFDAECZ)
- 2026-08-28T13:35:00Z a-root: restored the acceptance contract recorded on GitHub issue #813 after adoption produced an empty section; tracked systemic migration in #875.
- 2026-08-28T13:36:10Z accepted by a-root
- 2026-08-28T13:36:10Z closed WITHOUT verification — no --verify command was given
- 2026-08-28T13:36:10Z completed by a-root
- 2026-08-28T13:52:57Z a-root: Landing policy override: mode=pr base=main (event 01M149BYK31SG9V75JPJA47XYP)
- 2026-08-28T13:52:57Z a-root: Integrated via PR https://github.com/mlnomadpy/dacli/pull/865 at merge commit 551b97cd1deac675fe1ca2cd5f3e9742d30439fd into main (generation 0) (event 01M149C729DPY5RXQENCFTQGMW)
## Verification Evidence
{"command":"GOCACHE=/tmp/dacli-520-accept-cache go test ./... -count=1","exit_code":0,"duration_ms":66710,"artifact_hash":"sha256:284baf997be098102462fdf68dae1b1764bf4ab29886cba22cd2a93ea3c5f8c3","verifier":"a-root","branch":"dacli/520-agent-report-integrate-skips-a-second-pr-for-an-active-task-after-an-earlier-pr","commit_sha":"5ed712d8ebb404d04fb37c5fe565b9147e72c566"}
