---
id: f-mcp-schemas-now-have-executable-additive-compatibility-fixtures
kind: note
note_kind: finding
created: 2026-08-13T22:00:14Z
created_by: a-fixer-sn8j7p
about: "[[435]]"
severity: major
---
# MCP schemas now have executable additive compatibility fixtures
internal/mcp/schema_compat_test.go enumerates every published tool, reads internal/mcp/testdata/<tool>.json, recursively preserves stable fields/types, and tests that finish_task execution is not reached for missing, malformed, or unsupported schema_version; initial focused test failed because all fixtures and version validation were absent.
