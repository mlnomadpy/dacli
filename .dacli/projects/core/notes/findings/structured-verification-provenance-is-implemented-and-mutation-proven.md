---
id: f-structured-verification-provenance-is-implemented-and-mutation-proven
kind: note
note_kind: finding
created: 2026-08-13T21:25:12Z
created_by: a-fixer-3hxnxc
about: "[[432]]"
severity: major
---
# Structured verification provenance is implemented and mutation-proven
internal/store/verification.go persists JSON records with command, exit_code, duration_ms, artifact_hash, verifier, branch, and commit_sha; acceptance and planning reject missing command evidence. Mutation blanking ArtifactHash fails TestCloseRecordsVerificationEvidence with 'verification evidence missing artifact hash'. gofmt, go vet, focused tests, and go test -count=1 ./... pass; golangci-lint is unavailable (command not found).
