---
id: t-01M0D2KPGZZMYYSVSHNB8NS2T9
kind: task
created: 2026-08-19T13:15:45Z
created_by: a-root
owner: a-root
github:
  issue: 726
  repo: mlnomadpy/dacli
estimate: "{optimistic: 5, probable: 8, pessimistic: 13}"
---
# Prevent dacli push from replaying obsolete remote history after a local rebase
## Context
Adopted from GitHub issue #726.

## Symptom

A task branch was correctly rebased onto current `origin/main`, and `git diff origin/main...HEAD` contained only documentation/skill files. The remote task branch still pointed to the pre-rebase history. Running `dacli push 475` succeeded, but rewrote the local branch by replaying the obsolete remote commits. PR #716 then contained 18 already-merged source files from tasks 465, 474, and 470.

Observed recovery: rebase onto `origin/main` again, verify the merge base and three-dot diff, then use an exact `git push --force-with-lease=<remote-ref>:<observed-oid>`.

## Suspected cause

The non-fast-forward PushSync path appears to treat the remote task branch as work to preserve/replay, even when the local branch has deliberately rebased onto a newer trunk and the remote-only commits are patch-equivalent to commits already on trunk. This can silently contaminate an existing PR while reporting only `pushed <branch>`.

## Acceptance
- [x] A repository fixture reproduces: remote task branch based on old trunk, its code commits already merged/cherry-picked to current trunk, and a local task branch rebased onto current trunk with only a new docs commit.
- [x] `dacli push <task>` never replays those obsolete remote commits into the local branch or PR diff.
- [x] If safe synchronization is ambiguous, push refuses with exit 3 and names the exact lease-protected recovery instead of mutating history.
- [x] A successful rebased-branch push preserves `merge-base(local, origin/main) == origin/main` and the three-dot diff remains limited to the intended task files.
- [x] Output distinguishes an ordinary fast-forward push from any history-rewriting/lease-protected operation.
- [x] Mutation evidence makes the regression test fail when obsolete remote commits are replayed.
## Log
- 2026-08-19T14:08:52Z claimed by a-maintainer-4243q0
- 2026-08-19T14:23:32Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/745 (event 01M0D6CF55G6RP56VGCPSYV4T4)
- 2026-08-19T14:30:29Z accepted by a-root
- 2026-08-19T14:30:29Z verified by `GOCACHE=/tmp/dacli-accept-484 go test ./internal/gitx ./internal/features/vcs` (exit 0) in branch main at a0ed6bc — proves that tree builds, not that the work is in trunk
- 2026-08-19T14:30:29Z deliverable: dacli/484-prevent-dacli-push-from-replaying-obsolete-remote-history-after-a-local-rebase is merged into main
- 2026-08-19T14:30:29Z completed by a-root
## Verification Evidence
{"command":"GOCACHE=/tmp/dacli-accept-484 go test ./internal/gitx ./internal/features/vcs","exit_code":0,"duration_ms":803,"artifact_hash":"sha256:9223dc19b74d7d70fd64780950f939ca2ab4d794023c5047d032197ff7eaa177","verifier":"a-root","branch":"main","commit_sha":"a0ed6bcd92d6d3eea5cc1b00fcce110e9adf29a9"}
{"command":"GOCACHE=/tmp/dacli-accept-484 go test ./internal/gitx ./internal/features/vcs","exit_code":0,"duration_ms":760,"artifact_hash":"sha256:9223dc19b74d7d70fd64780950f939ca2ab4d794023c5047d032197ff7eaa177","verifier":"a-root","branch":"main","commit_sha":"a0ed6bcd92d6d3eea5cc1b00fcce110e9adf29a9"}
