---
id: f-claim-scope-gap-task-051-claimed-insight-spm-brief-gates-committed-internal
kind: note
note_kind: finding
created: 2026-07-22T16:40:57Z
created_by: a-root
severity: major
---
# Claim-scope gap: task 051 (claimed insight/spm/brief/gates) committed internal/store/calibration.go — outside its claim — yet E2's claim-scoped commit didn't block it, causing a merge conflict with 048. Two causes: (a) my brief put a store-file perf fix in a non-store-claimed task (author error), and (b) E2's enforcement didn't catch the out-of-claim file. Harden E2 to refuse/warn louder, and scope perf fixes to the package that owns the file
