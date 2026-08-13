---
id: 01KZYJHTZTYMWVM4TGHCPRMSFS
kind: event
schema_version: 1
event_kind: commit
created: 2026-08-13T22:05:45Z
created_by: a-fixer-sn8j7p
about: "[[t-01KZYA0JZ7EST3H42M5HTXNW21]]"
origin: agent
applied: true
checksum: sha256:e77416523d96166c50c078a249c72234156a08373f1da8306c7cb4e244c9fa52
---
6149670 435: version MCP schemas and pin additive compatibility

Red: TestSchemaVersionIsValidatedBeforeExecution accepted malformed and unsupported versions and ran the executor; TestStableToolSchemasMatchGoldenFixtures reported every fixture missing.

Verification: gofmt -l ., go vet ./..., go test ./..., and git diff --check pass. golangci-lint unavailable locally (command not found).
role: fixer
