---
id: f-verify-discarded-persisted-runtime-doctor-ro-evidence
kind: note
note_kind: finding
created: 2026-08-13T11:43:09Z
created_by: a-fixer-ge5keg
about: "[[411]]"
severity: major
---
# Verify discarded persisted runtime doctor RO evidence
internal/features/execution/verify.go loaded store.LoadRuntime declaration-only state, so sandboxFor refused a doctor-verified runtime as sandbox probe unknown; TestVerifyLoadsPersistedRuntimeROProbe reproduced the refusal before the fix.
