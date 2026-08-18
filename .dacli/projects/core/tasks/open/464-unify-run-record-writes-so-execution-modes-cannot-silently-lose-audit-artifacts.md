---
id: t-01M0AEG5AQPVJTH41MJNFRGSSX
kind: task
created: 2026-08-18T12:45:49Z
created_by: a-root
owner: a-root
priority: must
github:
  issue: 688
  repo: mlnomadpy/dacli
estimate: "{optimistic: 5, probable: 8, pessimistic: 13}"
---
# Unify run-record writes so execution modes cannot silently lose audit artifacts
## Context
Adopted from GitHub issue #688.

## Problem

Run-record durability is inconsistent across execution paths. `spawn` uses a helper that warns when `brief.md` or `invocation.txt` cannot be written, but `supervise`, `verify`, usage/result capture, provider outcome capture, kill/finalize, and detached outcomes still use unchecked `_ = os.WriteFile(...)` calls. A run can therefore execute or be finalized while the durable record silently omits the prompt, invocation, verdict, provider state, usage, kill marker, or terminal outcome.

For a system whose product is an auditable record, this is a correctness defect. The older task 024 fixed only the original spawn helper; later execution paths reintroduced the failure family.

## Design

Introduce one run-record writer with atomic replace semantics and explicit artifact criticality. Prompt, invocation, process identity, and terminal outcome failures must stop or visibly fail finalization. Optional usage/provider enrichment may warn, but the warning must be recorded in a durable diagnostic channel rather than only stderr. All execution modes use the same writer.



## Evidence

Unchecked writes remain at `internal/features/execution/execution.go:1453`, `:1460`, `:1479`, `:1494`, `:2108`, `:2111`, `:2832`, `:3072`, `:3100`, `:3133`, and `internal/features/execution/verify.go:128-151` (line numbers at audit time).

## Acceptance
- [ ] Spawn, supervise, verify, detached finalization, kill, provider-policy recording, and usage/result capture use one tested run-record writer.
- [ ] Brief, invocation, process identity, and terminal outcome artifacts have a documented fail-closed policy and cannot be silently omitted.
- [ ] Best-effort artifacts have a documented policy and a durable warning that `runs show` surfaces.
- [ ] Writes are atomic and do not expose truncated outcome/invocation files to concurrent readers.
- [ ] Fault-injection tests cover each execution mode with an unwritable or rename-failing run directory and assert the public command/result behavior.
- [ ] Finalization remains exactly once and does not append a success event when its terminal record failed.
- [ ] Mutation evidence removes the writer error propagation and makes the regression fail.
- [ ] Focused execution tests, race tests, and `go test ./...` pass.
## Log
