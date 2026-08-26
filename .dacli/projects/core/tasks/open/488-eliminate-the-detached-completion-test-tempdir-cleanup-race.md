---
id: t-01M0D4SN9N7MP3A02J76JZ32KW
kind: task
created: 2026-08-19T13:53:58Z
created_by: a-root
owner: a-root
github:
  issue: 737
  repo: mlnomadpy/dacli
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
---
# Eliminate the detached-completion test TempDir cleanup race
## Context
Adopted from GitHub issue #737.

## Reproduction

During root verification of an unrelated ghmirror-only branch, `go test ./...` failed in an untouched package:

```text
--- FAIL: TestDetachedCompletionDoesNotEquateUnobservablePIDWithExit (1.35s)
    testing.go:1464: TempDir RemoveAll cleanup: unlinkat .../TestDetachedCompletionDoesNotEquateUnobservablePIDWithExit.../003: directory not empty
```

The exact test then passed five consecutive runs, and `go test -p 1 ./...` passed. This indicates a timing-sensitive writer/process survives past test return when packages run with normal parallelism.

## Acceptance
- [ ] A stress or synchronization regression reproduces a writer/process touching the TempDir after the test body returns.
- [ ] The test and production helper explicitly join/reap every spawned writer/process before TempDir cleanup.
- [ ] `go test -race ./internal/features/execution -run TestDetachedCompletionDoesNotEquateUnobservablePIDWithExit -count=25` passes.
- [ ] Normal parallel `go test ./...` repeated in CI does not reproduce the cleanup race.
- [ ] Mutation evidence removes the join and makes the stress regression fail.
## Log
- 2026-08-26T13:36:48Z claimed by a-fixer-5hgvyg
