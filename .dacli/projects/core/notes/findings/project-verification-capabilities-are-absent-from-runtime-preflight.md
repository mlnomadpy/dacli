---
id: f-project-verification-capabilities-are-absent-from-runtime-preflight
kind: note
note_kind: finding
created: 2026-08-27T11:01:26Z
created_by: a-root
about: "[[509]]"
severity: major
origin: internal/features/execution/preflight.go:103
---
# Project verification capabilities are absent from runtime preflight
runtime doctor proves binary/version, grant sandbox, and startup transport; preflightIssues receives only runtime, role, grant, executable, and context. Neither path consumes the project's persisted Verification.Commands, so startup compatibility is incorrectly treated as sufficient for Gradle. Recommended design: introduce provider-neutral execution requirements (beginning with local coordination socket), infer them from the adopted/resolved verification workflow, persist capability state/provenance per runtime+grant, and fail closed when required capability is unsupported or unverified. Provider-specific probe mechanics belong behind the adapter behavioral-preflight strategy; scheduling consumes only generic capability states. Add a deterministic fake adapter that starts and writes successfully but refuses local socket bind, with macOS/Linux fixtures and no Android SDK.
