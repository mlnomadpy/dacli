---
id: t-01M0AEG5K7JF96HV0RJ5K17NJN
kind: task
created: 2026-08-18T12:45:49Z
created_by: a-root
owner: a-root
priority: should
github:
  issue: 686
  repo: mlnomadpy/dacli
estimate: "{optimistic: 8, probable: 13, pessimistic: 21}"
---
# Add command-scoped snapshots and enforce mature-workspace performance budgets
## Context
Adopted from GitHub issue #686.

## Problem

The markdown store is intentionally simple, but mature-workspace read paths repeatedly parse the same files. On the current dogfood workspace (461 tasks, 1,125 notes, 876 events, 202 runs), measured results on Apple M5 Pro include:

```text
BenchmarkBriefAssemble       ~120 ms/op   20.7 MB/op   96k allocs/op
BenchmarkNextReadPath         ~77 ms/op   10.7 MB/op   40k allocs/op
BenchmarkFindTaskLoop10      ~120 ms/op   41.3 MB/op  176k allocs/op
BenchmarkTaskIndexLoop10      ~13 ms/op    4.7 MB/op   25k allocs/op
BenchmarkEventlogListAll    ~35-44 ms/op   7.6 MB/op   36k allocs/op
```

The benchmark suite also still carries `BenchmarkGhmirrorListNotesPerTask`, a 12.8-second/1.95-GB reproduction of an O(tasks × notes) implementation that production already fixed by hoisting the scan. This makes the suite useful as archaeology but not as an enforceable performance contract.

## Design

Add an explicit command/cycle-scoped read snapshot that indexes tasks, notes, events, roles, and other immutable-for-the-operation inputs once. Mutation boundaries invalidate or rebuild the relevant snapshot; do not add a long-lived cache whose freshness is implicit. Refactor orchestration and brief assembly around loaded values so pure ranking/rendering can be benchmarked separately from filesystem I/O. Convert benchmarks into thresholds or relative guards on generated fixtures.

## Acceptance
- [ ] A command/cycle-scoped snapshot API has explicit freshness and invalidation semantics and never survives a mutation that affects its results.
- [ ] Orchestration resolves repeated task references from one index per stable phase rather than full-tree `FindTask` walks.
- [ ] Brief assembly receives one loaded event/task/note view and separates filesystem loading from pure selection/rendering.
- [ ] Generated-fixture benchmarks cover 100/400/1600 tasks and events and report linear scaling for scans and sublinear repeated lookup after indexing.
- [ ] Relative benchmark tests demonstrate repeated 10-ref resolution is materially faster than ten `FindTask` walks without machine-specific absolute timing assumptions.
- [ ] Obsolete fixed-defect benchmarks are moved to regression documentation or replaced by production-shape benchmarks.
- [ ] A performance budget is documented for the dogfood scale and a larger reference scale, including allocations.
- [ ] Correctness tests prove fresh sibling events and task transitions are visible at the next documented snapshot boundary.
- [ ] `go test -race` for store/eventlog/orchestration/brief paths and `go test ./...` pass.
## Log
