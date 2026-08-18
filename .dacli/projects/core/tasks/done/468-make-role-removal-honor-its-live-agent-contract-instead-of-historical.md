---
id: t-01M0AETPE835JWHHS5GA5RE4AW
kind: task
created: 2026-08-18T12:51:34Z
created_by: a-root
owner: a-root
priority: must
github:
  issue: 690
  repo: mlnomadpy/dacli
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
---
# Make role removal honor its live-agent contract instead of historical retirement flags
## Context
Adopted from GitHub issue #690.

## Reproduction

Task 463 attempted to retire obsolete provider/bootstrap roles after confirming no live run held them. `dacli role rm codex-maintainer` refused because 16 historical agent files still name the role, even though their runs are finished and `dacli agents` reports no live holder.

The public help and `RemoveRole` comment promise refusal while a **live** agent holds a role. The implementation instead checks `!a.Retired`, while WIP routing already uses `holdsWIPSlot`/process state to distinguish finished from live agents. Finished identities that were not explicitly retired therefore make a role permanent.

## Risk

Roles define execution capability. An obsolete or over-privileged role must be retractable once no process is using it. Requiring hundreds of historical identities to be manually retired makes roster cleanup unsafe and blocks the provider-neutral role migration in #689.

## Design

Use the same durable liveness definition for role removal and WIP accounting. Refuse for a live process and for a newly minted identity that has never run; allow removal when every holder is retired or has a terminal run and no live process. Fail closed if agent/run state cannot be read. Keep history attributable: removing a role definition must not delete agent or run records.



## Manual workaround

Retire every historical identity that names the role, one at a time, before removing it. This is impractical in the dogfood workspace and can accidentally retire a genuinely live holder if the operator reconstructs liveness incorrectly.

## Acceptance
- [x] A public-command regression creates a role, runs an agent under it to terminal completion without explicitly retiring the identity, and proves `role rm` succeeds.
- [x] A live run holding the role still causes exit 3 and the refusal names the live child/run.
- [x] A minted-but-never-run identity still blocks removal until retired, preserving the reserved WIP slot contract.
- [x] An explicitly retired identity does not block removal.
- [x] Unreadable agent or run state fails closed rather than certifying that no live holder exists.
- [x] Removing the role leaves historical agent/run records intact and `agent show`/`runs show` remain attributable even when the current role definition is absent.
- [x] The liveness predicate is shared with WIP accounting rather than duplicated with different semantics.
- [x] Mutation evidence, focused store/teamops tests, race tests, and `go test ./...` pass.
## Log
- 2026-08-18T12:52:08Z claimed by a-maintainer-ytrsg6
- 2026-08-18T13:33:36Z accepted by a-root
- 2026-08-18T13:33:36Z verified by `env GOCACHE=/tmp/dacli-468-final GOTMPDIR=/tmp go test ./...` (exit 0) in branch main at 8b71890 — proves that tree builds, not that the work is in trunk
- 2026-08-18T13:33:36Z deliverable: dacli/468-make-role-removal-honor-its-live-agent-contract-instead-of-historical is merged into main
- 2026-08-18T13:33:36Z completed by a-root
## Verification Evidence
{"command":"env GOCACHE=/tmp/dacli-468-accept GOTMPDIR=/tmp go test ./...","exit_code":0,"duration_ms":74501,"artifact_hash":"sha256:be911d2c164ab1b3ff4404df5e8649993281a14b6f908ed8819cb9cc739dc026","verifier":"a-root","branch":"main","commit_sha":"f441613cbac2d94a70d083fd51109cb6a99cbaab"}
{"command":"env GOCACHE=/tmp/dacli-468-final GOTMPDIR=/tmp go test ./...","exit_code":0,"duration_ms":80744,"artifact_hash":"sha256:2704e1dd08d32a1de31a3ad4110aaa4c41b82c9bab8a83bdba1dfe88cdefa592","verifier":"a-root","branch":"main","commit_sha":"8b71890591111579575a679f205f0249d26af645"}
