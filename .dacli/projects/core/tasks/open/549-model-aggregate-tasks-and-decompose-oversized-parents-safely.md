---
id: t-01M147RA6982P2FQXN64RZFPG4
kind: task
created: 2026-08-28T13:08:11Z
created_by: a-root
owner: a-root
github:
  issue: 866
  repo: mlnomadpy/dacli
estimate: "{optimistic: 8, probable: 13, pessimistic: 21}"
---
# Model aggregate tasks and decompose oversized parents safely
## Context
Adopted from GitHub issue #866.

## Parent

Part of #864.

## Observed symptom

A parent task with open implementation children can remain independently schedulable when no explicit dependency edges connect it to those children. The scheduler may then assign the oversized parent instead of its ready children, duplicating scope and defeating decomposition.

## Objective

Add an explicit aggregate/milestone task kind whose progress and terminal eligibility derive from its children, plus an atomic preview/apply workflow for repairing ambiguous parent/child graphs.

## Required behavior

- Aggregate tasks are never directly assigned to implementers while they have open implementation children.
- Aggregate readiness, progress, blockers, and completion are derived from stable child IDs and typed dependencies.
- Ordinary parent/child hierarchy remains descriptive unless the operator explicitly chooses aggregate semantics.
- Detect an independently schedulable parent whose children cover its implementation scope and propose an atomic dependency/kind repair.
- For oversized leaf tasks, propose—not silently create—a WBS containing checkable acceptance criteria, estimates, minimal path claims, and typed edges.



## Non-goals

- Inferring that every parent is an aggregate.
- Letting hierarchy substitute for explicit cross-project dependencies.
- Automatically decomposing work merely to satisfy a model-capacity limit.

## Manual workaround today

Operators manually add dependency edges from parent to children and ensure the parent is not selected.

## Acceptance
- [ ] A fixture with one aggregate parent and four ready children schedules only eligible children and reports the parent as aggregate progress.
- [ ] A non-aggregate parent remains schedulable, preserving backward-compatible descriptive hierarchy.
- [ ] Aggregate completion refuses while any required child is open, blocked, unverified, or unlanded under project policy.
- [ ] Repair preview and apply consume one versioned immutable plan and refuse if the graph changes between them.
- [ ] Proposed decomposition has stable IDs/references, acceptance criteria, estimates, path claims, and an acyclic typed dependency graph; no task is created without explicit apply authority.
- [ ] Critical-path, `next`, WBS, GitHub projection, and JSON views agree on aggregate state.
- [ ] Mutation tests fail when the aggregate parent is returned as directly assignable or can close before a required child.
## Log
