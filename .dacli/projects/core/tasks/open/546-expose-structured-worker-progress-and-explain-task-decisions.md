---
id: t-01M147RA0JHR6H2JN3H83RBAEH
kind: task
created: 2026-08-28T13:08:11Z
created_by: a-root
owner: a-root
github:
  issue: 869
  repo: mlnomadpy/dacli
estimate: "{optimistic: 8, probable: 13, pessimistic: 21}"
depends_on: "[t-01M146BA62817V08T9P6D6REKT, t-01M146BA26TEH86YE16XHDZKGY]"
---
# Expose structured worker progress and explain task decisions
## Context
Adopted from GitHub issue #869.

## Parent

Part of #864. Reuse the reconciliation projection in #856 rather than creating a second status model.

## Observed symptom

Live workers often remain at `thinking / TRANSCRIPT-ACTIVE` with a stale tail while actually editing, testing, committing, or preparing results. Scheduling and blocker explanations require combining agents, logs, worktree status, git history, tasks, dependencies, PRs, and CI manually.

## Objective

Expose structured worker progress and an `explain <task>` projection derived from durable/reconciled evidence.



## Non-goals

- Streaming private chain-of-thought.
- Treating transcript text as the only source of execution phase.
- High-cardinality hosted telemetry.

## Manual workaround today

Operators inspect `agents --tail`, run logs, worktree diffs, git history, tasks, critical path, and GitHub separately.

## Acceptance
- [ ] Live status reports phase/current command category, elapsed command time, last durable activity, changed paths, last commit, token/cost availability, PR/check state, next transition, and required operator action when observable.
- [ ] Every field names its source and observed time; stale or unavailable data is explicit rather than replaced with an optimistic phase.
- [ ] `dacli explain <task> --json` states scheduling rank, critical-path/slack contribution, blockers, aggregate/parent state, selected or rejected roles with capability/capacity reasons, relevant claims, landing state, and exact safe next action.
- [ ] Text status and JSON explain render the same typed projection shared with reconciliation where facts overlap.
- [ ] Polling remains bounded and avoids expensive process-tree/GitHub probes when cached evidence is still within a documented freshness window.
- [ ] Fixtures cover an editing worker, long-running test, silent/dead worker, awaiting-owner event, awaiting-merge PR, and external-state unknown.
- [ ] Mutation tests fail when stale activity is rendered as current or a rejected role disappears from explanation.
## Log
- 2026-08-28T13:32:48Z dependency edit by a-root (event 01M1495C79PB9FNNG4Z06JM36S)
