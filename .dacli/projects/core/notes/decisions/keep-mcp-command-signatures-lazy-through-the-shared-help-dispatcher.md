---
id: d-keep-mcp-command-signatures-lazy-through-the-shared-help-dispatcher
kind: note
note_kind: decision
created: 2026-08-19T12:03:51Z
created_by: a-maintainer-6w2h0v
about: "[[t-01M0AF65RDNBEX2SEF9JC5RTMZ]]"
github:
  issue: 723
  repo: mlnomadpy/dacli
---
# Keep MCP command signatures lazy through the shared help dispatcher
## Chose
Keep MCP command signatures lazy through the shared help dispatcher
## Rejected
Append every command Usage to the always-sent MCP tools/list description
## Because
The full catalog measured 20,727 bytes and grows with the CLI; a bounded table-derived discovery description plus per-command --help preserves one contract without recurring schema cost.
