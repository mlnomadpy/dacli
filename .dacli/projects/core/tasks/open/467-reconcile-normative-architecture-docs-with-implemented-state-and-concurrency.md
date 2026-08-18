---
id: t-01M0AEG5QK6WDSJQBZTG7Z8JWW
kind: task
created: 2026-08-18T12:45:49Z
created_by: a-root
owner: a-root
priority: should
github:
  issue: 685
  repo: mlnomadpy/dacli
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
---
# Reconcile normative architecture docs with implemented state and concurrency
## Context
Adopted from GitHub issue #685.

## Problem

The normative design documents contradict the shipped architecture:

- `DESIGN.md` says there is no shared mutable state and therefore no lock, while the store now has sequence/task/file locks, queue/stage transition locks, loop journals, runtime cooldown state, worktree state, and run records.
- `docs/ARCHITECTURE.md` calls `brief` an L4 pure engine that must not read disk, then later classifies it as an entity; `brief.Assemble` directly reads store, eventlog, prompts, risks, glossary, and notes.
- `DESIGN.md` still labels runtimes and GitHub/templates as specification-only even though all are implemented and central to the command surface.
- The project record still says the goal is to implement every planned stub and its codebase map reports 141 Go files; the repository now contains 330 Go files and no longer has that backlog shape.

Because assembled briefs include this project record and agents are told normative docs win over code, this drift actively misdirects implementation and review.

## Design

Write one current architectural truth: distinguish append-only contention avoidance from the small set of serialized shared transitions; distinguish pure policy engines from I/O assembly services; document which state is authoritative, derived, advisory, or recovery-critical; and make implementation status explicit. Generate volatile inventory data or add drift checks instead of hand-maintaining counts.



## Evidence

Measured 2026-08-18: 330 Go files, 87,989 Go lines, 461 task files, 1,125 notes, 876 events, and 202 run directories. `dacli project show core` still reports 141 Go files and the original stub-completion goal.

## Acceptance
- [ ] `DESIGN.md`, `docs/ARCHITECTURE.md`, `docs/DIAGRAMS.md`, and package docs agree on current layers and feature-slice boundaries.
- [ ] The concurrency model enumerates every shared mutable state family, its writer/lock/atomicity rule, and its failure policy.
- [ ] The persistence model labels canonical objects, append-only events, recovery journals, advisory snapshots, runtime cooldowns, and regenerable projections.
- [ ] The brief package is either split into pure assembly plus I/O loading or documented honestly as an entity/application service; the normative layer rule and architecture tests agree.
- [ ] Implemented runtimes, templates/gates, GitHub projection, loop, and control-plane bridge are no longer labeled specification-only.
- [ ] The core project goal, success criteria, and codebase map are refreshed to the current repository.
- [ ] A test or doctor check detects stale generated inventory and high-value normative contradictions such as “no locks” when lock files/APIs exist.
- [ ] All Markdown links resolve and the documented command/exit/JSON contracts are checked against the current CLI.
## Log
