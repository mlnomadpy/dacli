---
id: t-01KY849P3WJC88S88C7QHDSWKK
kind: task
created: 2026-07-23T18:37:38Z
created_by: a-root
owner: a-root
github:
  issue: 75
  repo: mlnomadpy/dacli
---
# loop --pr --auto stalls under strict branch protection (mid-cycle record push strands every fixer PR)
## Context
Adopted from GitHub issue #75.

## What I hit

With `strict` branch protection on `main` (require branches up-to-date before merge), **every fixer PR ended up `BEHIND` and its queued auto-merge stalled**, even at `--width 1` where only one PR is ever in flight.

## Root cause

Within a single cycle the sequence is:
1. fixer branches off `main@T0`, implements, opens PR, `dacli pr --auto` queues auto-merge
2. the loop closes the record and runs `ship --no-accept --no-integrate --push` — which **commits a `.dacli` record and pushes it to `main@T1`**

Now the fixer's PR is behind `main` by the loop's own record commit, so under `strict` GitHub won't auto-merge until the branch is updated. The loop advances `main` mid-cycle and strands the PR it just created.

## Impact

Auto-merge — the whole hands-off premise — stalls on every cycle. I had to `gh pr update-branch` each PR, and ultimately relax to `strict:false` to get clean self-merging. The irony: the record push is **data-only** (`.dacli`, never code), so a "behind" code PR can never actually conflict with it — the `strict` gate is protecting against a conflict that structurally cannot happen here.

## Suggested direction

Decouple the loop's record persistence from the PR-landing path so it doesn't advance `main` mid-cycle. Options:
- Don't `--push` the record commit each cycle; batch record pushes at sprint end (or let them ride a later cycle), keeping `main` stable while a fixer PR is in flight; or
- Route the `.dacli` record through the fixer's PR / a dedicated low-frequency commit rather than a per-cycle direct push; or
- Document that `dacli loop --pr --auto` wants `strict:false` (or repo "auto-update branch" enabled), and detect `strict` at loop start to warn.

## Note

This also blocks `--width > 1`: parallel fixer PRs each fall behind the others' merges under `strict`, so true parallel sprints aren't reliable without one of the above.

## Acceptance
## Log
