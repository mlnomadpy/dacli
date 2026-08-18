---
id: t-01M0AGCX64GETGKKBBJK5SKG7D
kind: task
created: 2026-08-18T13:18:59Z
created_by: a-root
owner: a-root
github:
  issue: 693
  repo: mlnomadpy/dacli
---
# Bind command verification provenance to the caller's actual worktree
## Context
Adopted from GitHub issue #693.

## Symptom

Running `dacli task check 468 --all --verify "... go test ./..."` from linked worktree `codex/690-role-removal-recovery2` at `a1c1485` executed the suite successfully in that checkout, but persisted `branch: main` and `commit_sha: f441613`. The record therefore identifies an artifact other than the one the command tested.

## Reproduction

1. Create or enter a linked Git worktree whose branch/HEAD differ from the main checkout.
2. From that worktree run `dacli task check <ref> --n <n> --verify "go test ./..."`.
3. Inspect the task Verification Evidence JSON.
4. Observe that branch/SHA name the shared workspace root, not the caller worktree.

## Suspected cause

`internal/store/verification.go:RunVerification` derives branch/SHA from `w.Root` and also sets `exec.Command.Dir = w.Root`. `workspace.Find` intentionally redirects `.dacli` records to the main checkout, but verification needs two locations: shared record root and the caller code checkout (`ctx.Cwd`). Collapsing them makes provenance false and can test the wrong tree.

## Risk

Verification can certify main while acceptance is being checked for unlanded task-branch code. That defeats artifact binding and creates the exact failure class dacli is designed to prevent: the record disagrees with reality.

## Manual workaround

Run verification separately in the intended worktree, record its actual `git branch --show-current` and `git rev-parse HEAD` in a finding/PR, and do not rely on the stored Verification Evidence provenance.

## Design

Pass an explicit execution/provenance directory from the command context into the verification layer. Keep task persistence on `w.Root`, but execute and resolve Git identity from the caller checkout. Reject or explicitly mark unknown provenance when the directory cannot be resolved; never silently substitute main.

## Acceptance criteria

- A public-command regression invokes task-check/accept from a linked worktree whose branch and HEAD differ from main and proves the verification command runs in that worktree.
- Persisted branch and commit SHA exactly match that linked worktree.
- The shared task record is still updated in the main `.dacli` workspace.
- A non-Git execution directory records honest unknown Git provenance without breaking command evidence.
- Mutation evidence proves substituting `w.Root` makes the worktree regression fail.
- Focused acceptance/planning/store tests, race tests, and `go test ./...` pass.

## Acceptance
## Log
