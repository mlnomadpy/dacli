---
id: d-084-filed-task-149-burn-alert-per-run-population-mismatch-as-the-single-highest
kind: note
note_kind: decision
created: 2026-07-26T15:26:54Z
created_by: a-q2y900ts5s
about: [[084]]
---
# 084: filed task 149 (burn alert per-run population mismatch) as the single highest-value evidence-based change
## Chose
084: filed task 149 (burn alert per-run population mismatch) as the single highest-value evidence-based change
## Rejected
Filing the governor window CEILING not being persisted (loop/*-governor.txt stores window_spent/window_start but not window_tokens), which limits loop status and the burn Windows view to showing spend with nothing to threshold it against
## Because
The window-ceiling gap is pure observability: the loop's own control flow is unaffected because the ceiling comes from flags each restart (orchestration.go:119), and loop status already labels the spent value correctly (orchestration.go:266,179) -- so it is moderate at best. The burn-alert population mismatch (burn.go:121-163 counts every run for Rate vs burn.go:266-274/calibration.go:217 counting only completing non-verify runs for Ceiling) is a CORRECTNESS defect in a feature whose entire stated purpose is to yell accurately at 1.5x: cheap same-day review/verify runs dilute PerRun and produce a FALSE-NEGATIVE that masks a genuine 2x implementer overspend. A wrong alert threshold defeats the feature, so fixing the yell's accuracy outranks adding a ceiling readout. Also considered but rejected the burn ULID-decode 50-vs-48-bit overflow in ulidTime (burn.go:211-224): real ulid.At output never overflows, so it is a non-defect.
