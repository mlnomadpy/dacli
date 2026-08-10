---
id: d-filed-the-checkallacceptance-section-rewrite-data-loss-335-confirmed-the
kind: note
note_kind: decision
created: 2026-08-10T17:33:26Z
created_by: a-go-auditor-d451f3
about: "[[303]]"
---
# Filed the CheckAllAcceptance section-rewrite data loss (335); confirmed the estimate-degradation and gate-read-error findings are already fixed at HEAD
## Chose
Filed the CheckAllAcceptance section-rewrite data loss (335); confirmed the estimate-degradation and gate-read-error findings are already fixed at HEAD
## Rejected
Re-filing the CPM/routing unestimated-degradation finding (now handled by sizeUnestimated, orchestration.go:686) or the gates read-error pass (now propagated, orchestration.go:1512-1514), or filing the loop-close deadlock (already task 312)
## Because
I verified the prior cycle's recorded findings against current code before picking: sizeUnestimated now sizes the batch before routing, and advanceStages/gates.Advance now propagate and log read errors instead of swallowing them, so those are stale. CheckAllAcceptance remains a genuine, code-cited record-integrity defect (store.go:314 replaces the Acceptance section with mdstore.RenderCheckboxes, which mdstore.go:677-704 flattens to bare checkboxes) that no task queues and no test pins. It is the #1 audit class — a close path silently rewriting the record it claims to only tick 'in place' (store.go:296). It is latent for this repo's pure-checkbox dogfood tasks but live for any user with prose or nested acceptance criteria, and the fix is small and well-scoped (in-place line flip, not a rewrite).
