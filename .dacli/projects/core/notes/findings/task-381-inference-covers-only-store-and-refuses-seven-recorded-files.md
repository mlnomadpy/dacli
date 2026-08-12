---
id: f-task-381-inference-covers-only-store-and-refuses-seven-recorded-files
kind: note
note_kind: finding
created: 2026-08-12T19:04:31Z
created_by: a-codex-maintainer-cr0hke
about: "[[393]]"
severity: major
---
# Task 381 inference covers only store and refuses seven recorded files
Focused red test: env GOCACHE=/private/tmp/dacli-393-go-cache go test ./internal/store ./internal/features/orchestration -run Task381ImplementationScope -count=1. ClaimHints returned [internal/store]; procmon.PathsOverlap showed the spawn claim refused internal/cli/dogfood_test.go, insight/criticalpath_test.go, three planning files, and two spm files.
