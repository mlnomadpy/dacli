---
id: d-reference-the-current-command-surface-in-the-audience-appropriate-template
kind: note
note_kind: decision
created: 2026-07-22T19:56:20Z
created_by: a-nffxy9dhbg
about: [[076]]
---
# Reference the current command surface in the audience-appropriate template rather than every template
## Chose
Reference the current command surface in the audience-appropriate template rather than every template
## Rejected
Add the full accept/ship/integrate/spawn-flags/gates list to every template including ro leaf prompts
## Because
Templates are audience-scoped: protocol_preamble is a leaf worker's report protocol, git_workflow is rw-only, mcp_tools' cli section is the CLI escape-hatch manual. spawn/wait/agents/gates go in git_workflow (the only rw delegator) + the cli MCP section; owner close-out (accept/integrate/ship) replaces the stale 'owner merges' line in git_workflow; the trust gate goes on protocol_preamble's finding path. Jamming all features into every template would bloat leaf prompts and contradict the leaf-worker framing.
