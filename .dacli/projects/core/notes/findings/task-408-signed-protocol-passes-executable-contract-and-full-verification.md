---
id: f-task-408-signed-protocol-passes-executable-contract-and-full-verification
kind: note
note_kind: finding
created: 2026-08-13T14:35:46Z
created_by: a-fixer-z4s0m6
about: "[[408]]"
severity: major
---
# Task 408 signed protocol passes executable contract and full verification
contracts/controlplane/v1/contract_test.go:35 verifies seven closed versioned schemas and required envelope fields; :65 classifies six distinct golden outcomes. Mutation evidence: removing producer_version failed TestVersionedSchemasAndEnvelope at contract_test.go:51 with envelope missing property producer_version. gofmt -l . emitted nothing; GOCACHE=/private/tmp/dacli-408-gocache go vet ./... passed; /Users/tahabsn/go/bin/golangci-lint v2.12.2 reported 0 issues; GOCACHE=/private/tmp/dacli-408-gocache go test ./... passed.
