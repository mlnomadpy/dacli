---
id: f-cloudsync-tamper-test-mutation-fails-at-signature-boundary
kind: note
note_kind: finding
created: 2026-08-13T16:13:53Z
created_by: a-codex-maintainer-sg0bxk
about: "[[425]]"
severity: major
---
# Cloudsync tamper test mutation fails at signature boundary
internal/cloudsync/client_test.go:178 requires ErrInvalidSignature and unchanged state plus empty inbox. Mutation: forcing Verify to return true failed TestReceiveFixturesAndSecurityChecks/tampered.json because it got ErrInvalidPayload instead of ErrInvalidSignature.
