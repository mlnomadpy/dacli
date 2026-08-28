---
id: t-01M1493JBGSSHWDXAVAY5W7B9E
kind: task
created: 2026-08-28T13:31:48Z
created_by: a-root
owner: a-root
github:
  issue: 878
  repo: mlnomadpy/dacli
estimate: "{optimistic: 5, probable: 8, pessimistic: 13}"
depends_on: "[t-01M146BA62817V08T9P6D6REKT, t-01M146B9TFB8Y6CX6FMMK91J9Q]"
---
# Reconcile and compact obsolete event journals without rewriting history
## Context
Adopted from GitHub issue #878.

## Parent

Extracted from #871. Depends on canonical classification in #856 and should share immutable-plan safety with cleanup #862.

## Observed symptom

Long-lived workspaces accumulate pending journal records, terminal-task events, missing targets, malformed documents, and obsolete proposals. `sync` cannot apply every kind, while manual editing violates the append-only audit model.

## Objective

Add audited event-journal reconciliation and bounded compaction without rewriting historical facts.



## Non-goals

- Deleting history solely because it is old.
- Treating every unapplied event as an error.
- Compacting live-run evidence needed for recovery.

## Manual workaround today

Operators inspect event Markdown directly, dismiss records one by one, and avoid cleanup because provenance and recoverability are unclear.

## Acceptance
- [ ] A read-only plan classifies pending actionable mailbox events, complete journal events, terminal/missing targets, superseded proposals, malformed records, and unknown/unreadable evidence.
- [ ] Apply consumes the same immutable plan and resolves obsolete mailbox work through append-only dismissal/supersession records; it never silently edits or deletes an original event.
- [ ] Safe compaction creates a verifiable snapshot/index plus content hashes before any recoverable archival move; durable task/decision/finding provenance remains queryable.
- [ ] Unknown, malformed, contested, or externally referenced records are preserved with an exact manual action.
- [ ] `status`, `events pending`, `sync`, reconciliation, and loop recovery agree on what remains actionable after apply.
- [ ] Reruns are idempotent and crash/restart fixtures cover interruption after plan, dismissal, snapshot, and archival move.
- [ ] Retention is policy-configured by evidence class, not age alone, and dry-run reports byte/count impact without mutation.
- [ ] Mutation tests fail when original history is rewritten or an actionable proposal disappears.
## Log
- 2026-08-28T13:32:48Z dependency edit by a-root (event 01M1495CR7Z15YM8EH00PB40DN)
