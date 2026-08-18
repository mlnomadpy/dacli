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
- [x] Spawn, supervise, verify, detached finalization, kill, provider-policy recording, and usage/result capture use one tested run-record writer.
- [x] Brief, invocation, process identity, and terminal outcome artifacts have a documented fail-closed policy and cannot be silently omitted.
- [x] Best-effort artifacts have a documented policy and a durable warning that `runs show` surfaces.
- [x] Writes are atomic and do not expose truncated outcome/invocation files to concurrent readers.
- [x] Fault-injection tests cover each execution mode with an unwritable or rename-failing run directory and assert the public command/result behavior.
- [x] Finalization remains exactly once and does not append a success event when its terminal record failed.
- [x] Mutation evidence removes the writer error propagation and makes the regression fail.
- [x] Focused execution tests, race tests, and `go test ./...` pass.
## Log
- 2026-08-18T14:14:42Z claimed by a-maintainer-zppm9n
- 2026-08-18T14:36:48Z accepted by a-root
- 2026-08-18T14:36:48Z verified by `env GOCACHE=/tmp/dacli-post-wave-gocache GOTMPDIR=/tmp go test ./...` (exit 0) in branch main at e82449a — proves that tree builds, not that the work is in trunk
- 2026-08-18T14:36:48Z deliverable: dacli/464-unify-run-record-writes-so-execution-modes-cannot-silently-lose-audit-artifacts is merged into main
- 2026-08-18T14:36:48Z completed by a-root
- 2026-08-18T15:14:58Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/701 (event 01M0AMAD9VE7B92H90TXMR87JH)
## Verification Evidence
{"command":"env GOCACHE=/tmp/dacli-post-wave-gocache GOTMPDIR=/tmp go test ./...","exit_code":0,"duration_ms":3033,"artifact_hash":"sha256:07db5a7f494d86162ce19dd327d134bb8ac313296b8ab5376f50a81125baa384","verifier":"a-root","branch":"main","commit_sha":"e82449acf21e69bc44b33e31a57caafd9b45c7c8"}
