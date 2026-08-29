# Read-path performance budgets

dacli's markdown store favors inspectability over a hidden process cache. Read
performance therefore depends on explicit command/cycle snapshots: load files
once at the start of a stable phase, resolve references from the loaded index,
and discard or invalidate that snapshot before a mutation. The next command or
cycle loads a fresh view, which is also when sibling events become visible.

Run the production-shape and generated-fixture suite with:

```sh
go test -run '^$' -bench 'Benchmark(Brief|Generated|FindTask|TaskIndex|Eventlog|ScaleCreateTask)' -benchmem ./internal/perfbench
```

The generated suite covers 100, 400, and 1,600 tasks and events. Scan time and
allocations should scale approximately linearly with fixture size (a 4x input
should remain below 6x, allowing filesystem noise). Ten reference resolutions
after one task-index load must use less than one third of the allocations of ten
independent `FindTask` walks; a normal test enforces this relative, machine-
independent guard.

Reference budgets are review triggers, not nanosecond CI assertions:

| Scale | Brief load + render | Task snapshot + 10 lookups | Event scan | Allocations |
|---|---:|---:|---:|---:|
| Dogfood: 500 tasks, 1,200 notes, 1,000 events, 250 runs | 150 ms | 20 ms | 55 ms | brief 25 MB / 120k; task phase 6 MB / 35k; events 10 MB / 50k |
| Large: 1,600 tasks, 4,000 notes, 3,200 events, 800 runs | 500 ms | 65 ms | 180 ms | brief 85 MB / 400k; task phase 20 MB / 115k; events 32 MB / 165k |

Absolute timings vary by filesystem and CPU, so CI enforces the allocation
ratio and correctness boundaries while benchmark review checks the reference
budgets. A budget regression needs either a measured explanation or a change
that restores the production shape; archaeology for already-fixed quadratic
paths belongs in commit/issue history rather than an executable benchmark.

Task sequence allocation is the write-path exception to full markdown scans.
A locked, durable acceleration state records the next sequence together with
task-directory, tombstone, and Git-ref observations. Matching observations make
creation independent of backlog size; any manual file, removal, ref, malformed
state, or crash drift rebuilds from canonical filenames, tombstones, and Git
history before another number is issued. `BenchmarkScaleCreateTask` covers 100,
400, 1,600, and 6,400 existing tasks and should keep allocations per creation
approximately flat.
