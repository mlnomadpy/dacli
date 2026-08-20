---
id: f-mcp-usage-parity-implementation-eagerly-injects-the-entire-cli-catalog-into
kind: note
note_kind: finding
created: 2026-08-19T11:59:42Z
created_by: a-root
about: "[[470]]"
severity: major
---
# MCP usage parity implementation eagerly injects the entire CLI catalog into every tools list
Commit 2d29299 makes mcp.CommandDescription append every command Usage to the generic cli tool description. That turns a help-drift fix into a recurring context/schema tax for every MCP client and conflicts with #707 prompt-budget goals. Preserve one command table and the invariant, but keep tools/list compact: expose full signatures lazily through the existing cli help path or a dedicated discovery response, and test that MCP routes discovery to the same table instead of embedding all ~120 synopses in the tool description. Measure the description size before/after and set a bounded regression.
