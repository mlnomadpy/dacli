---
id: d-anchor-documentation-drift-checks-to-the-manually-maintained-mcp-registry
kind: note
note_kind: decision
created: 2026-08-12T20:12:31Z
created_by: a-codex-maintainer-p44wb5
about: "[[367]]"
github:
  issue: 538
  repo: mlnomadpy/dacli
---
# Anchor documentation drift checks to the manually maintained MCP registry
## Chose
Anchor documentation drift checks to the manually maintained MCP registry
## Rejected
Generate MCP tools from the CLI command table or parse implementation source files in a test
## Because
The shipped architecture intentionally keeps a curated small MCP catalog; an in-package test can enumerate the actual tool table and check the public support claims without coupling documentation verification to Go source text.
