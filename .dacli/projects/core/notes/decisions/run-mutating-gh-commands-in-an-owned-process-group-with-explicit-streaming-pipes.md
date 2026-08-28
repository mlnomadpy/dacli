---
id: d-run-mutating-gh-commands-in-an-owned-process-group-with-explicit-streaming-pipes
kind: note
note_kind: decision
created: 2026-08-28T09:06:38Z
created_by: a-maintainer-fb7kb1
about: "[[t-01M1068MTFPQ6YFVQG204M2EX4]]"
---
# Run mutating gh commands in an owned process group with explicit streaming pipes
## Chose
Run mutating gh commands in an owned process group with explicit streaming pipes
## Rejected
Keep os/exec writer pipes and kill only the direct gh leader
## Because
explicit pipes let leader exit, process-tree reconciliation, transcript streaming, and stream drain occur in the required order; direct-child cancellation leaves descendants and the sequence lease observable after return
