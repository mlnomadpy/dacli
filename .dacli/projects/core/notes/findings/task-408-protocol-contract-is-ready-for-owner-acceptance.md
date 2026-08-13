---
id: f-task-408-protocol-contract-is-ready-for-owner-acceptance
kind: note
note_kind: finding
created: 2026-08-13T14:26:24Z
created_by: a-fixer-6nxvsp
about: "[[408]]"
severity: major
---
# Task 408 protocol contract is ready for owner acceptance
contracts/controlplane/v1 now contains the seven closed v1 payload schemas, signed envelope, compatibility/retention/offline-replay policy, and six executable golden outcomes. Mutation evidence: removing producer_version made TestVersionedSchemasAndEnvelope fail with 'envelope missing property producer_version'. Green evidence: gofmt -l . emitted nothing; GOCACHE=/private/tmp/dacli-408-gocache go vet ./... passed; golangci-lint v2.12.2 reported 0 issues; GOCACHE=/private/tmp/dacli-408-gocache go test ./... passed. task check was correctly refused because owner a-root alone may check boxes.
