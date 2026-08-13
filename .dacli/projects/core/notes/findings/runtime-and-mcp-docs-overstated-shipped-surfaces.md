---
id: f-runtime-and-mcp-docs-overstated-shipped-surfaces
kind: note
note_kind: finding
created: 2026-08-12T20:12:31Z
created_by: a-codex-maintainer-p44wb5
about: "[[367]]"
severity: moderate
---
# Runtime and MCP docs overstated shipped surfaces
internal/features/execution/execution.go:62-121 ships exactly five presets (claude-code, claude-code-rw, generic-exec, codex, codex-rw), while docs/RUNTIMES.md named Gemini, opencode, and mock as shipped. internal/mcp/tools.go:122-405 manually defines fifteen core tools plus cli, including check_task; docs claimed fourteen generated schemas.
