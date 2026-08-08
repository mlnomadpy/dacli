---
id: d-documented-accept-ship-integrate-wait-agents-kill-logs-github-project-catalog
kind: note
note_kind: decision
created: 2026-07-22T22:00:26Z
created_by: a-gx269dxyzs
about: [[080]]
---
# Documented accept/ship/integrate/wait/agents/kill/logs/github project/catalog/calibrate/spawn under the mcp_tools cli escape-hatch section rather than adding Tier-1 schemas
## Chose
Documented accept/ship/integrate/wait/agents/kill/logs/github project/catalog/calibrate/spawn under the mcp_tools cli escape-hatch section rather than adding Tier-1 schemas
## Rejected
promote each admin command to its own MCP Tier-1 tool schema
## Because
MCP.md ties the tiered surface to the 14 core verbs an agent touches between claim and done; admin verbs are rare/human-run and a 50-schema catalog is the per-agent context tax the design refuses elsewhere — the cli hatch is where they belong, so the fix is completing that section, not widening the tier
