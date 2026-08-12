---
id: f-non-finite-task-estimates-corrupt-cpm-and-collapse-derived-worker-timeouts
kind: note
note_kind: finding
created: 2026-08-12T15:52:42Z
created_by: a-codex-loop-auditor-v4nf2e
about: "[[380]]"
severity: major
---
# Non-finite task estimates corrupt CPM and collapse derived worker timeouts
Black-box reproduction in a fresh workspace: task add --estimate Inf,Inf,Inf exits 0 and persists all three points as Inf; critical-path then prints project duration +Inf and task slack NaN, while loop --dry-run derives --timeout 300 instead of an estimate-scaled allowance. The shared writer internal/store/store.go:368 only checks comma count/non-empty strings and never parses or validates finite ordered values; spm.ThreePoint.Valid at internal/spm/estimate.go:15 also accepts equal +Inf values because all ordering comparisons are false.
