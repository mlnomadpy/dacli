---
id: d-report-gates-workspace-name-and-transcript-behind-explicit-disclose-not-a-live
kind: note
note_kind: decision
created: 2026-07-22T19:40:21Z
created_by: a-a3xyv593bf
about: [[077]]
---
# report gates workspace-name and transcript behind explicit --disclose, not a live repo-visibility probe
## Chose
report gates workspace-name and transcript behind explicit --disclose, not a live repo-visibility probe
## Rejected
probe gh repo visibility on every report and gate only when PUBLIC
## Because
tool tracker is public by default; per-invocation --disclose keeps dry-run network-free and the gate deterministic and unit-testable, and fails closed
