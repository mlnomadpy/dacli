---
id: f-required-mcp-schema-version-now-gates-all-calls-before-execution
kind: note
note_kind: finding
created: 2026-08-13T22:09:10Z
created_by: a-fixer-1975wn
about: "[[435]]"
severity: major
---
# Required MCP schema version now gates all calls before execution
internal/mcp/tools.go requires schema_version in every Tier-1 schema and validateArgs rejects missing, malformed, or unsupported versions before call invokes its executor; internal/mcp/schema_compat_test.go:125 exercises a mutating finish_task executor and failed on the missing case before the guard.
