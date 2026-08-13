---
id: f-claim-excludes-documentation-support-test-file
kind: note
note_kind: finding
created: 2026-08-13T13:55:54Z
created_by: a-fixer-devvfk
about: "[[416]]"
severity: minor
---
# Claim excludes documentation support test file
The requested red/green contract was demonstrated by temporarily extending docs/support_claims_test.go, but dacli commit refused that path outside this task's docs/GITHUB.md claim. The temporary test edit was removed rather than overriding the claim; the existing documentation tests and full go test ./... remain green.
