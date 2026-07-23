---
id: t-01KY849P411MWTDSN21RE822GT
kind: task
created: 2026-07-23T18:37:38Z
created_by: a-root
owner: a-root
github:
  issue: 73
  repo: mlnomadpy/dacli
---
# loop review phase files near-duplicate tasks (no dedup vs open backlog)
## Context
Adopted from GitHub issue #73.

## What I hit

Over a long sprint run, the loop's review phase filed several **near-duplicate tasks**:
- `106` "charge idle-cycle review spawns to the token window" ≈ `108` "charge idle-cycle reviewer tokens to the --window-tokens budget"
- `109` "bound the three remaining unbounded git/gh subprocesses" ≈ `110` "give the last three unbounded git and gh subprocesses deadlines"

`109` was even filed with `[0/0]` acceptance (empty). I had to hand-dedupe with `dacli accept <ref> --force`.

## Root cause

The review phase spawns an auditor against the standing improvement task each cycle, but the auditor **does not see the current open backlog** — so it re-discovers and re-files an issue a prior cycle already queued. There's no dedup between "what the review wants to file" and "what's already open/recently-done."

## Impact

This is precisely the *diminishing-returns / churn* signal a self-governed perpetual loop should detect and stop on. Instead it manufactures redundant work; an unattended `--yolo` run would burn cycles building the same thing twice (and near-empty tasks like `109`).

## Suggested direction

- Pass the **open + recently-done task titles** into the review auditor's brief so it can skip already-queued work (dedup at file time).
- Or a lightweight post-filter: reject a newly-filed task whose title/So-that is a high-similarity match to an existing open task.
- Bonus: treat "review filed only duplicates for K cycles" as an additional **halt signal** for the governor — the honest "nothing new of value to do" stop condition, complementing the trunk-advance thrash guard.

## Acceptance
## Log
