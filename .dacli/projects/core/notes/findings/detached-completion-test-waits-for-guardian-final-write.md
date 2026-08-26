---
id: f-detached-completion-test-waits-for-guardian-final-write
kind: note
note_kind: finding
created: 2026-08-26T13:43:38Z
created_by: a-fixer-5hgvyg
about: "[[t-01M0D4SN9N7MP3A02J76JZ32KW]]"
severity: major
---
# Detached completion test waits for guardian final write
internal/features/execution/execruntime_test.go:53 now waits for runtime-exit.txt after recorder completion; TestAwaitDetachedCompletionWaitsForGuardianFinalWrite proves the delayed writer is joined. Mutation removing the awaitGuardianExitFile call fails at execruntime_test.go:93: detached completion returned before guardian's final TempDir write. Verified: GOCACHE=/private/tmp/dacli-go-cache go test -race ./internal/features/execution -run '^TestDetachedCompletionDoesNotEquateUnobservablePIDWithExit$' -count=25; gofmt -l .; go vet ./...; and go test ./... -count=2 passed. golangci-lint is unavailable in this environment (command not found).
