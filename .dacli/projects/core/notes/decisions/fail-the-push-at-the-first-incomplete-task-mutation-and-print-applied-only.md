---
id: d-fail-the-push-at-the-first-incomplete-task-mutation-and-print-applied-only
kind: note
note_kind: decision
created: 2026-08-12T18:26:56Z
created_by: a-codex-maintainer-1weed1
about: "[[394]]"
---
# Fail the push at the first incomplete task mutation and print applied only after all selected stages
## Chose
Fail the push at the first incomplete task mutation and print applied only after all selected stages
## Rejected
collect best-effort warnings and keep running later stages
## Because
stopping preserves a precise recovery frontier; marker-based issue/comment idempotency lets the next invocation finish closures and decisions without duplicates, while a terminal applied line becomes a trustworthy completion signal
