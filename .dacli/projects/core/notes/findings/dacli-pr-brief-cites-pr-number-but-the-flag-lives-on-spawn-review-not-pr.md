---
id: f-dacli-pr-brief-cites-pr-number-but-the-flag-lives-on-spawn-review-not-pr
kind: note
note_kind: finding
created: 2026-07-22T18:08:20Z
created_by: a-5zfa3xx3z5
about: [[059]]
severity: minor
---
# dacli pr Brief cites --pr-number but the flag lives on spawn --review, not pr
internal/features/vcs/lifecycle.go:29 the 'pr' command Brief mentions '--pr-number' (and the earlier report cross-referenced it), but cmdPR (lifecycle.go:~136-173) never reads f.Get('pr-number') — verdicts are posted by branch name (gh pr review <branch>). The only --pr-number reader is spawn --review at internal/features/execution/execution.go:1154. Documenting pr with --pr-number would be wrong; RUNTIMES.md keeps --pr-number under spawn --review only.
