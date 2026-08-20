---
id: d-generate-mcp-cli-signatures-from-the-aggregated-command-table
kind: note
note_kind: decision
created: 2026-08-19T11:55:39Z
created_by: a-fixer-yd1rff
about: "[[t-01M0AF65RDNBEX2SEF9JC5RTMZ]]"
github:
  issue: 722
  repo: mlnomadpy/dacli
---
# Generate MCP cli signatures from the aggregated command table
## Chose
Generate MCP cli signatures from the aggregated command table
## Rejected
Maintain a separate MCP synopsis list
## Because
The generic MCP cli tool and CLI help must expose identical command contracts, so a second list would recreate the drift fixed here.
