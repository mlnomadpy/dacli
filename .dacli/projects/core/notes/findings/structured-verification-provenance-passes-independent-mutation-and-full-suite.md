---
id: f-structured-verification-provenance-passes-independent-mutation-and-full-suite
kind: note
note_kind: finding
created: 2026-08-13T21:41:28Z
created_by: a-fixer-hchbmq
about: "[[432]]"
severity: major
---
# Structured verification provenance passes independent mutation and full-suite checks
At e85b5a4, blanking ArtifactHash made go test -count=1 ./internal/features/acceptance -run '^TestCloseRecordsVerificationEvidence$' fail at evidence_test.go:67 with 'verification evidence missing artifact hash'; after restoration, focused store/acceptance/planning tests, gofmt -l ., go vet ./..., and GOCACHE=/private/tmp/dacli-432-test-cache go test -count=1 ./... passed. golangci-lint was unavailable (command not found).
