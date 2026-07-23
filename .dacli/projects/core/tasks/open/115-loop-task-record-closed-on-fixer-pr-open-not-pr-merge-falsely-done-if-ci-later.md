---
id: t-01KY849P3ZTV2X5SKZY4GG2ATP
kind: task
created: 2026-07-23T18:37:38Z
created_by: a-root
owner: a-root
priority: could
github:
  issue: 74
  repo: mlnomadpy/dacli
---
# loop: task record closed on fixer PR-open, not PR-merge (falsely 'done' if CI later fails)
## Context
Adopted from GitHub issue #74.

## What I hit

In the loop's self-PR land model (`--pr`), `runCycle` closes each built task's record with `accept --force` right after `wait` — i.e. the moment the fixer **opens its PR**, not when the PR **merges**. Under the default `--pr --auto` path the merge is asynchronous (GitHub merges on green CI, seconds-to-minutes later, and *may never* if CI fails).

So a task is marked `done` while its PR is still pending — and if that PR later fails CI and never merges, the task is **falsely closed** with the work not on trunk.

## Why this is subtle

The thrash-guard progress signal is already correct here — it measures real trunk advancement, so a never-merged PR shows `landed=0` and can still halt the loop. But the **task record** diverges from trunk reality: the backlog says done, main disagrees. The only thing that recovers it is the review phase re-discovering the un-landed work and re-filing it — which is indirect and not guaranteed.

Related but distinct from the spawn-refused/failed case (already handled): here the spawn *succeeded* and a PR *opened*, it just hasn't (or won't) merge.

## Impact

Silent backlog/trunk divergence on any CI-failing agent PR. In a longer unattended run this accumulates: "done" tasks whose code never landed.

## Suggested direction

Close the task record on **confirmed merge**, not on PR-open:
- Defer the `accept --force` to a subsequent cycle once the trunk marker confirms the task's PR merged (the loop already fetches/rev-lists trunk each cycle); or
- Reconcile against `gh pr view <branch> --json state` before closing; or
- Mark such tasks `active`/`in-review` rather than `done` until the merge lands, so the backlog never claims completion the trunk can't back.

## Acceptance
## Log
