---
id: f-mcp-command-catalog-inflated-every-tools-list-response
kind: note
note_kind: finding
created: 2026-08-19T12:03:51Z
created_by: a-maintainer-6w2h0v
about: "[[t-01M0AF65RDNBEX2SEF9JC5RTMZ]]"
severity: major
---
# MCP command catalog inflated every tools/list response
Measured commit 2d29299's MCP tools/list response at 20,727 bytes because internal/mcp/mcp.go appended every Usage synopsis; exact signatures can instead be fetched through the shared executor's per-command --help path.
