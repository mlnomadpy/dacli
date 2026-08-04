---
id: d-210-split-applyproposals-into-read-pendingproposals-consume
kind: note
note_kind: decision
created: 2026-08-04T10:17:49Z
created_by: a-maintainer-qy7k6x
about: "[[210]]"
---
# 210: split applyProposals into read (pendingProposals) + consume (markProposalsApplied), mark only after CloseTask succeeds
## Chose
210: split applyProposals into read (pendingProposals) + consume (markProposalsApplied), mark only after CloseTask succeeds
## Rejected
keep single applyProposals but move the whole call after CloseTask
## Because
the accept-log line 'accepted by X (applied N proposal(s))' is written into the task doc and flushed by CloseTask's own SaveTask, so the count must be known BEFORE the close; a read-now/consume-after split lets the log report the pending count while the durable MarkApplied is deferred until after the close lands
