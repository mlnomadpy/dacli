---
id: t-01M0CZANAQKP50AWEN2C6C8VXR
kind: task
created: 2026-08-19T12:18:23Z
created_by: a-root
owner: a-root
github:
  issue: 717
  repo: mlnomadpy/dacli
estimate: "{optimistic: 5, probable: 8, pessimistic: 13}"
---
# Add an audited dependency-edit command for adopted and existing tasks
## Context
Adopted from GitHub issue #717.

## Problem

`dacli github pull` can adopt an existing GitHub issue as a local task, but dependency edges are only accepted during `dacli task add`. Once a task exists, an operator or fresh agent cannot express the critical-path dependencies discovered during backlog refinement through the CLI. Editing task frontmatter by hand bypasses validation and the event record.

## Design direction

Add a task dependency mutation surface (for example `dacli task depend`) that resolves workspace/project-qualified references, supports the existing dependency types, validates cycles and ambiguity before writing, and records an audited event. It must work for GitHub-adopted and locally-created tasks without changing their GitHub mapping.

## Acceptance criteria

- A documented CLI command adds and removes dependency edges on an existing task, including one adopted by `dacli github pull`.
- The command accepts the dependency types already supported by task creation and preserves their round trip through task show, critical-path, next, and GitHub synchronization.
- Ambiguous references, missing tasks, self-dependencies, and dependency cycles fail closed without partially writing the task.
- Read-only agents propose the mutation for owner synchronization; unauthorized writes return the documented policy-refusal exit.
- Tests cover add, remove, idempotent replay, ambiguity, cycles, project-qualified references, and adopted-task GitHub mapping preservation.
- Mutation proof demonstrates a cycle/validation guard change makes the focused test fail for the intended reason.
- CLI and MCP help derive the exact signature from the shared command table.

## Acceptance
- [x] A documented CLI command adds and removes dependency edges on an existing task, including one adopted by `dacli github pull`.
- [x] The command accepts the dependency types already supported by task creation and preserves their round trip through task show, critical-path, next, and GitHub synchronization.
- [x] Ambiguous references, missing tasks, self-dependencies, and dependency cycles fail closed without partially writing the task.
- [x] Read-only agents propose the mutation for owner synchronization; unauthorized writes return the documented policy-refusal exit.
- [x] Tests cover add, remove, idempotent replay, ambiguity, cycles, project-qualified references, and adopted-task GitHub mapping preservation.
- [x] Mutation proof demonstrates a cycle/validation guard change makes the focused test fail for the intended reason.
- [x] CLI and MCP help derive the exact signature from the shared command table.
## Log
- 2026-08-20T08:25:03Z claimed by a-maintainer-d7gr0n
- 2026-08-20T09:05:52Z accepted by a-root
- 2026-08-20T09:05:52Z verified by `GOCACHE=/tmp/dacli-task479-gocache GOMODCACHE=/tmp/dacli-task479-gomodcache go test ./internal/store ./internal/features/planning ./internal/eventlog` (exit 0) in branch main at cbb16bd — proves that tree builds, not that the work is in trunk
- 2026-08-20T09:05:52Z deliverable: dacli/479-add-an-audited-dependency-edit-command-for-adopted-and-existing-tasks is merged into main
- 2026-08-20T09:05:52Z completed by a-root
- 2026-08-20T09:08:12Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/757 (event 01M0F685MTX2RJ2DNX2AP4VWEY)
## Verification Evidence
{"command":"GOCACHE=/tmp/dacli-task479-gocache GOMODCACHE=/tmp/dacli-task479-gomodcache go test ./internal/store ./internal/features/planning ./internal/eventlog","exit_code":0,"duration_ms":562,"artifact_hash":"sha256:123d6ca4731f37b5963593469faa56ca810e7d8704451345139b1485d0feedb2","verifier":"a-root","branch":"main","commit_sha":"cbb16bd6415eb05ed51479f082eaccc0699a44fa"}
{"command":"GOCACHE=/tmp/dacli-task479-gocache GOMODCACHE=/tmp/dacli-task479-gomodcache go test ./internal/store ./internal/features/planning ./internal/eventlog","exit_code":0,"duration_ms":505,"artifact_hash":"sha256:123d6ca4731f37b5963593469faa56ca810e7d8704451345139b1485d0feedb2","verifier":"a-root","branch":"main","commit_sha":"cbb16bd6415eb05ed51479f082eaccc0699a44fa"}
