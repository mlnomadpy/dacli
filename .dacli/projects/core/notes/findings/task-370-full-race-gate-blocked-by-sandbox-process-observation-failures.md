---
id: f-task-370-full-race-gate-blocked-by-sandbox-process-observation-failures
kind: note
note_kind: finding
created: 2026-08-12T15:48:25Z
created_by: a-codex-maintainer-3vy9w1
about: "[[370]]"
severity: moderate
---
# Task 370 full race gate blocked by sandbox process-observation failures
go test -race ./... passes orchestration and most packages but fails existing sandbox-sensitive tests: internal/cli E2E spawned worker exits 1; execution oversized detached prompt reads 0 and guardian identity is unobservable; procmon sees 0 processes / cannot read process start. Focused orchestration suite passes. Acceptance boxes remain unchecked under CONTRIBUTING's red-suite rule.
