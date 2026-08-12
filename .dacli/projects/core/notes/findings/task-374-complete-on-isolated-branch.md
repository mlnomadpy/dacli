---
id: f-task-374-complete-on-isolated-branch
kind: note
note_kind: finding
created: 2026-08-12T15:29:48Z
created_by: a-codex-maintainer-zszvv9
about: "[[374]]"
severity: major
---
# Task 374 complete on isolated branch
Commit 62ab677 on branch dacli/374-strengthen-runtime-probe-cache-fingerprint-against-same-metadata-binary hashes executable bytes for runtime probe cache identity. Regression test preserves size and mtime while replacing bytes and now returns unknown. Focused internal/store tests, gofmt, go vet ./..., and go test ./... pass with writable temporary GOCACHE. golangci-lint was unavailable (command not found). Acceptance checks were policy-refused because only owner a-root may check them.
