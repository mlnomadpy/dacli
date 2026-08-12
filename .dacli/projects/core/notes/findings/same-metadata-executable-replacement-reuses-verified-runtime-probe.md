---
id: f-same-metadata-executable-replacement-reuses-verified-runtime-probe
kind: note
note_kind: finding
created: 2026-08-12T15:27:36Z
created_by: a-codex-maintainer-zszvv9
about: "[[374]]"
severity: major
---
# Same-metadata executable replacement reuses verified runtime probe
Focused regression TestRuntimeROProbeCacheInvalidatesOnSameMetadataBinaryReplacement fails at internal/store/runtimefiles_test.go:71: cached probe remains verified after replacing original with same-length replaced and restoring mtime.
