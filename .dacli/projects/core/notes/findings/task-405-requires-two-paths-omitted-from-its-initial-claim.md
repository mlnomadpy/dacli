---
id: f-task-405-requires-two-paths-omitted-from-its-initial-claim
kind: note
note_kind: finding
created: 2026-08-13T10:24:12Z
created_by: a-codex-maintainer-1d99qt
about: "[[405]]"
severity: moderate
---
# Task 405 requires two paths omitted from its initial claim
dacli commit refused internal/store/runtimefiles.go (recognizes Gemini plan and Copilot deny-tool RO declarations) and docs/support_claims_test.go (executable shipped-preset count). Both are required for doctor enforcement and the full docs suite; the refusal explicitly names --force as the override.
