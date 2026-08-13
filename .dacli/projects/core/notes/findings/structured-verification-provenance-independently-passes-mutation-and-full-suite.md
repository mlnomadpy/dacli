---
id: f-structured-verification-provenance-independently-passes-mutation-and-full-suite
kind: note
note_kind: finding
created: 2026-08-13T21:31:02Z
created_by: a-fixer-dq7v5k
about: "[[432]]"
severity: major
---
# Structured verification provenance independently passes mutation and full-suite checks
Commit e85b5a4 persists command, exit_code, duration_ms, artifact_hash, verifier, branch, and commit_sha in internal/store/verification.go. Temporarily blanking ArtifactHash made TestCloseRecordsVerificationEvidence fail with 'verification evidence missing artifact hash'; after restoration, gofmt -l ., go vet ./..., focused tests, and go test -count=1 ./... passed. golangci-lint could not run because the binary is not installed.
