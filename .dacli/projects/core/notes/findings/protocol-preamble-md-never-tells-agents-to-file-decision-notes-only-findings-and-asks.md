---
id: f-protocol-preamble-md-never-tells-agents-to-file-decision-notes-only-findings-and-asks
kind: note
note_kind: finding
created: 2026-07-21T23:09:25Z
created_by: a-zjtzasqfb4
about: [[t-01KY3EKR201B2Y30GWGQR42CNC]]
---
# protocol_preamble.md never tells agents to file decision notes, only findings and asks
internal/prompts/tpl/protocol_preamble.md is the one place every spawned child (rw and ro) learns how to report. It documents note add finding and ask explicitly, plus task check/done for rw, but never mentions note add decision -- even though mcp_tools.md's add_note section (internal/prompts/tpl/mcp_tools.md:35) calls the rejected-alternative out as 'the valuable part' of a decision note and docs/PROMPTS.md's constraint log shows decisions are a load-bearing artifact (see d-app-layer-is-feature-sliced style entries). An agent that only ever reads protocol_preamble (the rw/ro reporting contract) has no textual cue that decisions are a thing it should record, so architecturally-relevant choices with rejected alternatives risk landing only as prose in a commit message instead of a structured decision note that future briefs pick up. Fix: add one bullet parallel to the finding bullet, e.g. 'When you choose an approach over a real alternative: note add decision ... --rejected ... --because ...'.
