# Go system-design playbook

## Architecture

Keep dependencies flowing from shared packages to entities to isolated feature
slices to the app layer. When two features need a rule, move the rule downward
instead of importing sideways. Keep command metadata—usage, JSON support, and
mutation capability—declarative so registry-wide invariants can inspect it.

## Persistence and concurrency

Treat every durable file as protocol state. Use one lock for the whole
read-modify-write transaction, write a sibling temporary file, close/fsync when
durability requires it, and rename atomically. Never swallow errors for journals,
run records, permissions, or verification evidence. State transitions should be
idempotent and recoverable after every intermediate durable effect.

## Performance

Benchmark against a mature workspace, report time/bytes/allocations, and inspect
the call graph before caching. Prefer a command-scoped immutable snapshot and
indexes keyed by task, agent, run, and event over repeated directory walks.
Bound retained transcripts and caches. Preserve correctness under concurrent
writers and validate with `go test -race` on affected packages.
