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
- [x] A command/cycle-scoped snapshot API has explicit freshness and invalidation semantics and never survives a mutation that affects its results.
- [x] Orchestration resolves repeated task references from one index per stable phase rather than full-tree `FindTask` walks.
- [x] Brief assembly receives one loaded event/task/note view and separates filesystem loading from pure selection/rendering.
- [x] Generated-fixture benchmarks cover 100/400/1600 tasks and events and report linear scaling for scans and sublinear repeated lookup after indexing.
- [x] Relative benchmark tests demonstrate repeated 10-ref resolution is materially faster than ten `FindTask` walks without machine-specific absolute timing assumptions.
- [x] Obsolete fixed-defect benchmarks are moved to regression documentation or replaced by production-shape benchmarks.
- [x] A performance budget is documented for the dogfood scale and a larger reference scale, including allocations.
- [x] Correctness tests prove fresh sibling events and task transitions are visible at the next documented snapshot boundary.
- [x] `go test -race` for store/eventlog/orchestration/brief paths and `go test ./...` pass.
## Log
- 2026-08-20T08:03:44Z claimed by a-maintainer-dv3gq7
- 2026-08-20T08:23:43Z accepted by a-root
- 2026-08-20T08:23:43Z verified by `GOCACHE=/tmp/dacli-task466-root-cache go test -race ./internal/store ./internal/brief ./internal/features/orchestration -run 'TestTaskSnapshot|TestIndexedTenRef|TestViewBoundary' -count=1` (exit 0) in branch main at 1e49799 — proves that tree builds, not that the work is in trunk
- 2026-08-20T08:23:43Z deliverable: dacli/466-add-command-scoped-snapshots-and-enforce-mature-workspace-performance-budgets is merged into main
- 2026-08-20T08:23:43Z completed by a-root
- 2026-08-20T09:08:12Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/755 (event 01M0F3X881KNW14XGYX5PQCG5H)
## Verification Evidence
{"command":"GOCACHE=/tmp/dacli-task466-root-cache go test -race ./internal/store ./internal/brief ./internal/features/orchestration -run 'TestTaskSnapshot|TestIndexedTenRef|TestViewBoundary' -count=1","exit_code":0,"duration_ms":4639,"artifact_hash":"sha256:b1e3f1d4607adc26f8ecd9e340bf8bb1c588df911944039182237060f79167ff","verifier":"a-root","branch":"main","commit_sha":"1e49799840650c3613e857e450af3f1cdabcf4bd"}
{"command":"GOCACHE=/tmp/dacli-task466-root-cache go test -race ./internal/store ./internal/brief ./internal/features/orchestration -run 'TestTaskSnapshot|TestIndexedTenRef|TestViewBoundary' -count=1","exit_code":0,"duration_ms":3661,"artifact_hash":"sha256:048cbe1e1fecd59ca8597557ac3750c9ef7461393bf187f80c029106e7497606","verifier":"a-root","branch":"main","commit_sha":"1e49799840650c3613e857e450af3f1cdabcf4bd"}
