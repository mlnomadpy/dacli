---
id: t-01M146BA434CD4A9778E8BNJ61
kind: task
created: 2026-08-28T12:43:36Z
created_by: a-root
owner: a-root
github:
  issue: 857
  repo: mlnomadpy/dacli
estimate: "{optimistic: 5, probable: 8, pessimistic: 13}"
---
# Bind verification evidence to the immutable committed tree
## Context
Adopted from GitHub issue #857.

## Parent

Part of #855.

## Observed symptom

Structured verification evidence records `HEAD`, but a command can run against dirty uncommitted changes. The evidence can therefore name the parent commit even though a different working tree was tested. It also lacks a tree SHA, dirty-state assertion, tool versions, and GitHub check/run identifiers.

## Objective

Bind every acceptance-grade verification record to the exact immutable tree that was executed and make stale evidence mechanically rejectable.

## Required design

- Capture commit SHA and tree SHA before verification.
- Refuse acceptance-grade verification of a dirty tree unless an explicitly named non-acceptance advisory mode is used.
- Re-read commit/tree/dirty state after the command and reject evidence if the tested tree changed during execution.
- Record argv/working directory, exit code, duration, verifier identity, runtime/tool versions, and output artifact digest.
- Provide typed attachment points for GitHub workflow run/check IDs and explicitly skipped external evidence.
- Make acceptance/ship compare required evidence to the exact reviewed/landing head tree.



## Non-goals

- Replacing GitHub required checks.
- Treating an output hash as a source-tree hash.
- Automatically committing user changes.

## Suspected cause

`internal/store/verification.go` captures the current commit before command execution but does not bind or freeze the worktree tree.

## Manual workaround today

Commit first, verify a clean checkout, manually record the exact SHA, then confirm the PR head has not changed before acceptance.

## Acceptance
- [ ] A regression test shows the current dirty-tree/parent-SHA scenario is refused for acceptance-grade evidence.
- [ ] A test that mutates the worktree or advances `HEAD` while verification runs records no successful evidence and names the before/after mismatch.
- [ ] Successful evidence contains commit SHA, tree SHA, clean-state assertion, structured command/cwd, exit code, duration, verifier/runtime/tool versions, and output digest.
- [ ] Acceptance and ship refuse evidence whose tree differs from the canonical reviewed PR head, with an actionable message.
- [ ] GitHub check/run identifiers can be attached without treating an unobservable or skipped check as green.
- [ ] Backward compatibility for historical evidence is explicit: old evidence remains readable but cannot silently satisfy a new final-tree gate.
- [ ] The targeted regression fails when tree comparison or the dirty-tree guard is removed; full Go quality gates pass.
## Log
