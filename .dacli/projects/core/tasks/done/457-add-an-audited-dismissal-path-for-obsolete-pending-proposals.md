---
id: t-01KZZSD1K4YT88J0YYB5ZPD75R
kind: task
created: 2026-08-14T09:24:42Z
created_by: a-root
owner: a-root
github:
  issue: 670
  repo: mlnomadpy/dacli
priority: must
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
---
# Add an audited dismissal path for obsolete pending proposals
## Context
Adopted from GitHub issue #670.

Implementation and claim boundary: `internal/eventlog` owns durable event disposition and pending/read filtering; `internal/features/collab` owns the operator command and authorization orchestration; `internal/store` may consume the canonical dismissed predicate for reference checks. Preserve append-only event provenance and do not teach `task rm` to bypass unresolved references.

## Reproduction

After #667 / PR #669 merged, root recovery authorization for retired child-owned task 454 succeeded far enough to reach canonical reference safety:

    dacli task rm 454 --force

returns exit 3 because task 454 is referenced by two pending events:

- 01KZYZWQGSQ71JFFEPE2DN47AC-a-root-claim.md
- 01KZZJ8GRERMVKAMRQWJHT9QE9-a-root-block.md

Those proposals were diagnostic attempts by a-root before #667 existed. dacli sync cannot apply them because task 454 is owned by the retired audit child. There is no command to reject, dismiss, or withdraw an obsolete pending event. Deleting event files manually would violate the append-only collaboration record.

## Proven gap

eventlog supports pending and applied materialization, and sync intentionally leaves unauthorized or malformed proposals pending. The CLI exposes apply through sync/accept but no audited terminal disposition for a proposal the owner/root deliberately rejects. store.RemoveTask correctly treats the pending references as live, making a safe orphan cleanup impossible.

## Manual workaround

None through dacli. Manual record deletion is intentionally rejected as unsafe.

## Design

Add an append-only proposal dismissal/rejection operation. It must preserve the original event and record who dismissed it, when, and why; dismissed events must no longer fold into reads, be considered pending, or block canonical task removal. Authorization should mirror the action boundary: an event author may withdraw an unapplied proposal, and rw object owner/root may reject it when the target owner is retired or the proposal is otherwise unresolvable. Applied events cannot be dismissed as if they never happened.

## Acceptance
- [x] A command lists pending events with stable event IDs, actor, action, target, and authorization state so an operator can select the exact proposal.
- [x] An event author can withdraw their own unapplied proposal with a required reason.
- [x] The rw target owner can reject an unapplied proposal; rw root can reject one targeting a known retired child-owned orphan.
- [x] Read-only identities and unrelated siblings cannot dismiss another actor proposal.
- [x] Dismissal preserves the original event and adds an audited terminal disposition with actor, timestamp, and reason instead of deleting the event file.
- [x] Dismissed events are excluded from pending counts, read folding, sync, accept proposal consumption, and task reference checks.
- [x] Applied events cannot be dismissed; the command names the correct compensating workflow instead.
- [x] A regression reproduces task 454 cleanup: dismiss the root claim/block proposals, then task rm 454 --force succeeds through store.RemoveTask.
- [x] Repeated dismissal is idempotent and does not duplicate audit records.
- [x] A dismissal with corrupt integrity metadata does not suppress the original proposal or unblock canonical task removal, and is surfaced as an unreadable event.
- [x] Mutation evidence, eventlog/collab/store tests, and go test ./... pass.
## Log
- 2026-08-14T09:25:42Z claimed by a-maintainer-rmzh0s
- 2026-08-14T10:05:47Z accepted by a-root
- 2026-08-14T10:05:47Z verified by `GOCACHE=/tmp/dacli-457-accept GOMODCACHE=/tmp/dacli-457-gomodcache go test ./...` (exit 0) in branch main at 1a0ce7c — proves that tree builds, not that the work is in trunk
- 2026-08-14T10:05:47Z deliverable: dacli/457-add-an-audited-dismissal-path-for-obsolete-pending-proposals is merged into main
- 2026-08-14T10:05:47Z completed by a-root
- 2026-08-14T10:06:07Z accepted by a-root
- 2026-08-14T10:06:07Z verified by `GOCACHE=/tmp/dacli-457-accept GOMODCACHE=/tmp/dacli-457-gomodcache go test ./...` (exit 0) in branch main at 1a0ce7c — proves that tree builds, not that the work is in trunk
- 2026-08-14T10:06:07Z deliverable: dacli/457-add-an-audited-dismissal-path-for-obsolete-pending-proposals is merged into main
- 2026-08-14T10:06:07Z completed by a-root
- 2026-08-14T10:19:45Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/671 (event 01KZZTR9H515A9EFS7G52Z6VX6)
## Verification Evidence
{"command":"GOCACHE=/tmp/dacli-457-accept GOMODCACHE=/tmp/dacli-457-gomodcache go test ./...","exit_code":0,"duration_ms":61505,"artifact_hash":"sha256:3304df010704d2b71351c3e41504f138a51dad8b2577a00cc60dcb83dbd4daf8","verifier":"a-root","branch":"main","commit_sha":"1a0ce7cd7b626e1c86c53f8e34e47b08de69c080"}
{"command":"GOCACHE=/tmp/dacli-457-accept GOMODCACHE=/tmp/dacli-457-gomodcache go test ./...","exit_code":0,"duration_ms":34391,"artifact_hash":"sha256:3ee5aab807e8a3eaac1bc9ee0680298791d39f90cd7ddfc10ca50e775e1f5011","verifier":"a-root","branch":"main","commit_sha":"1a0ce7cd7b626e1c86c53f8e34e47b08de69c080"}
