---
id: f-codex-spawn-previously-authorized-launch-from-sandbox-evidence-alone
kind: note
note_kind: finding
created: 2026-08-20T09:15:01Z
created_by: a-maintainer-dxj9ch
about: "[[t-01M0F3795JGCAG6ZS3XVAGNS2J]]"
severity: major
---
# Codex spawn previously authorized launch from sandbox evidence alone
internal/features/execution/execution.go hydrated only RuntimeROProbe before sandboxFor and returned a launch plan without executing the Codex exec/app-server startup path; the new fixture in internal/features/execution/preflight_test.go reproduces Operation not permitted before records are minted.
