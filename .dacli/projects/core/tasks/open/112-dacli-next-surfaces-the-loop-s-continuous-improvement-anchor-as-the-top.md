---
id: t-01KY849P3RT5ZMH2SA9MXECGS7
kind: task
created: 2026-07-23T18:37:38Z
created_by: a-root
owner: a-root
github:
  issue: 77
  repo: mlnomadpy/dacli
---
# dacli next surfaces the loop's 'Continuous improvement' anchor as the top actionable task
## Context
Adopted from GitHub issue #77.

## What I hit

After the MUST tasks were done, `dacli next` reported the standing **"Continuous improvement: file the single highest-value evidence-based change"** task as its #1 actionable item.

That task is the loop's *review-phase anchor* — an auditor is spawned against it every cycle to file new work; it is never itself implementer work. The loop's own `readyTasks` correctly excludes any task whose title starts with `Continuous improvement`, but `dacli next` (the planning/MoSCoW view) has no such exclusion, so a human (or an agent reading `next`) is told to "work on" the anchor.

## Impact

Minor but confusing: the planning view and the loop's execution view disagree about what's actionable, and the top recommendation is a task nobody should pick up directly.

## Suggested direction

- Factor the "is this an implementable task vs. a loop-internal anchor" predicate into a shared helper and apply it in **both** `readyTasks` and `dacli next` (single source of truth), or
- Give the anchor a distinct `kind`/marker in front-matter and have `next` skip that kind, rather than matching on the title prefix in one place only.

## Acceptance
## Log
