---
id: t-01M146B9TFB8Y6CX6FMMK91J9Q
kind: task
created: 2026-08-28T12:43:36Z
created_by: a-root
owner: a-root
github:
  issue: 862
  repo: mlnomadpy/dacli
estimate: "{optimistic: 5, probable: 8, pessimistic: 13}"
depends_on: "[t-01M146BA62817V08T9P6D6REKT, t-01KZYNVFR96911X4D1V3D9T98Y]"
---
# Plan and apply safe repository cleanup from reconciled state
## Context
Adopted from GitHub issue #862.

## Parent and prerequisite

Part of #855. Depends on the canonical reconciliation/classification model. Coordinate with #647 rather than duplicating its worktree-prune parity fix.

## Objective

Add repository-wide cleanup planning that classifies material before any mutation:

```bash
dacli cleanup --project <project> --dry-run
dacli cleanup --project <project> --apply-safe
```

## Required classifications

- Protected/current/base branch.
- Dirty or untracked worktree.
- Unpushed commits.
- Open, merged, closed-unmerged, missing, or superseded PR.
- Task/run/claim ownership and terminal status.
- Safely pruneable generated run artifacts versus durable evidence.
- Unknown external state.

## Safety constraints

- Dry-run and apply must consume the same immutable plan; apply refuses if observed state changed.
- `--apply-safe` may perform only explicitly enumerated recoverable operations.
- Dirty, unpushed, open-PR, closed-unmerged, protected, ambiguous, or externally unknown material is never deleted automatically.
- Branch/worktree cleanup must retain enough audit evidence to explain what was removed and why.



## Non-goals

- Broad recursive deletion.
- Deleting user branches solely because they are old.
- Treating a finished agent as proof that its work landed.

## Manual workaround today

Operators combine worktree pruning, branch/PR inspection, task state, and run cleanup manually.

## Acceptance
- [ ] A fixture containing every required classification produces the same plan in text and versioned JSON.
- [ ] Dry-run writes nothing, and applying its plan after any branch/worktree/PR state change refuses as stale.
- [ ] Only merged, clean, pushed, non-protected material with terminal task/run evidence is eligible for automatic cleanup.
- [ ] Dirty, unpushed, open-PR, closed-unmerged, protected, ambiguous, and GitHub-unobservable cases are preserved with an actionable reason.
- [ ] Apply uses recoverable git/worktree operations where available and prints the exact removed refs/paths plus recovery evidence.
- [ ] Regression tests fail when dry-run/apply share no plan identity or when any protected fixture becomes eligible.
- [ ] #647 behavior is reused or superseded through the shared planner without two conflicting cleanup classifiers.
## Log
- 2026-08-28T12:44:47Z dependency edit by a-root (event 01M146DF1WFR4HVQBNJJYJR0A7)
