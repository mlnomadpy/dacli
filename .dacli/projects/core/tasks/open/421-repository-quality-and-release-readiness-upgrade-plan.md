---
id: t-01KZXS3QNDANVK83M26D3S8W7M
kind: task
created: 2026-08-13T14:41:08Z
created_by: a-root
owner: a-root
github:
  issue: 437
  repo: mlnomadpy/dacli
---
# Repository quality and release-readiness upgrade plan
## Context
Adopted from GitHub issue #437.

## Objective

Make dacli reproducibly release-ready and demonstrate that its autonomous engineering workflow is safe, measurable, and reliable on real repositories.

## Required work

- [ ] Reproduce and fix `TestExecRuntimeDetachedDeliversAnOversizedPrompt`; prove detached execution preserves large prompts without truncation.
- [ ] Make the Unix process-monitor tests deterministic across macOS and Linux, including process groups, recycled PIDs, and zombie handling.
- [ ] Ensure `go test ./...` passes from a clean checkout on every supported platform.
- [ ] Add race-detector, fuzz, and failure-injection jobs for orchestration, process management, Git operations, and event-log recovery.
- [ ] Add an end-to-end fixture repository that dacli plans, implements, reviews, tests, and ships without manual intervention.
- [ ] Measure completion rate, retry rate, wall time, token/cost budget, and human-intervention rate across repeatable scenarios.
- [ ] Document the trust/taint model, command execution boundaries, secret handling, branch protection, and rollback behavior.
- [ ] Explain agent-authored commit identities and how self-hosted changes are reviewed and attributed.
- [ ] Publish a stable CLI/MCP schema compatibility policy and migration notes.
- [ ] Produce tagged releases, checksums, SBOMs, and cross-platform binaries.

## Acceptance criteria

- [ ] CI is green from a clean checkout on Linux and macOS, including race-sensitive tests.
- [ ] One documented self-hosting case study is fully reproducible.
- [ ] Security and failure-recovery documentation covers every mutating surface.
- [ ] A tagged release can be installed and exercised using README commands only.

## Path-specific implementation map

### Execution and process safety

- [ ] In `internal/features/execution/execruntime_test.go`, retain the oversized-prompt regression and add boundary cases around pipe-buffer size, stdin closure, detached execution, cancellation, and partial writes.
- [ ] In `internal/features/execution/`, replace any single-write assumption with a write-all loop that propagates short writes and child startup failures; keep transport concerns separate from runtime lifecycle state.
- [ ] In `internal/procmon/`, introduce an OS adapter interface for process discovery, start-time identity, group sampling, signal escalation, and reaping. Put Linux `/proc` behavior and macOS behavior in build-tagged files rather than shared timing assumptions.
- [ ] Add `internal/procmon/testdata/` helper binaries that deliberately fork, ignore TERM, become zombies, and recycle process-group state; use readiness pipes instead of sleeps in tests.
- [ ] In `internal/features/orchestration/driver_test.go`, add crash/restart cases proving claimed work, budgets, and event state recover without duplicate agent execution.

### Durable orchestration and observability

- [ ] In `internal/eventlog/`, add checksummed event envelopes, schema versions, corruption detection, replay checkpoints, and migration tests for old logs.
- [ ] In `internal/features/queues/` and `internal/features/stagegate/`, add explicit idempotency keys, retry classification, dead-letter state, and audit events for every transition.
- [ ] In `internal/features/acceptance/` and `internal/gates/`, persist evidence objects containing command, exit code, duration, artifact hash, and verifier identity rather than a boolean pass/fail only.
- [ ] In `internal/features/insight/` and `internal/features/wscore/`, emit run metrics through a small internal metrics interface; add JSON export for completion, retry, failure class, wall time, and budget consumption.
- [ ] Add `internal/scenarios/` with deterministic fixture repositories and scenario definitions for feature work, regression repair, dependency failure, conflicting edits, and malicious instructions.

### Interfaces, security, and releases

- [ ] In `internal/mcp/`, version tool schemas, add golden JSON fixtures under `internal/mcp/testdata/`, and test backward compatibility and malformed input.
- [ ] In `internal/gitx/` and `internal/features/vcs/`, enforce repository-root containment, safe ref validation, clean worktree rules, and explicit refusal tests for destructive/broad targets.
- [ ] Extend `SECURITY.md` with the command trust boundary, environment filtering, credential handling, worktree isolation, prompt-injection model, and incident reporting.
- [ ] Add `docs/SELFHOSTING_CASE_STUDY.md` generated from one fixture run and `scripts/run-release-scenario.sh` to reproduce it.
- [ ] Extend `.github/workflows/` with Linux/macOS tests, `go test -race ./...`, fuzz smoke tests, dashboard UI tests, release builds, SBOM generation, checksums, and artifact verification.
- [ ] Keep `cmd/dacli/main.go` thin; new behavior should enter through tested packages under `internal/features/` and be exposed through both CLI and MCP contract tests.

## Acceptance
## Log
