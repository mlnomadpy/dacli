---
id: f-task-432-wrapped-exiterror-lint-correction-passes-local-verification
kind: note
note_kind: finding
created: 2026-08-13T21:44:48Z
created_by: a-fixer-qgqmdp
about: "[[432]]"
severity: major
---
# Task 432 wrapped ExitError lint correction passes local verification
internal/store/verification.go now uses errors.As for *exec.ExitError, eliminating the CI errorlint failure at the former line 53 while preserving nonzero exit codes. GOCACHE=/private/tmp/dacli-432-qgqmdp-cache go test -count=1 ./internal/store ./internal/features/acceptance ./internal/features/planning, gofmt -l ., go vet ./..., and go test -count=1 ./... passed. golangci-lint could not be rerun locally because the pinned executable is not installed (command not found).
