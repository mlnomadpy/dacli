---
id: d-central-json-dispatch-gate-command-json-flag-not-per-command-json-handling
kind: note
note_kind: decision
created: 2026-08-04T20:45:25Z
created_by: a-maintainer-7zg8zg
about: "[[291]]"
---
# Central --json dispatch gate + Command.JSON flag, not per-command json handling
## Chose
Central --json dispatch gate + Command.JSON flag, not per-command json handling
## Rejected
Hand-adding ctx.JSON handling to each of ~40 read commands, or a global emitter
## Because
A dispatch-layer gate (refuseUnsupportedJSON in cli.go, wired into both Main and the MCP executor) makes silent-ignore structurally impossible: a command either declares Command.JSON and honors the flag, or --json is refused exit-2 with a hint listing the honoring commands. This mirrors why Flags.Reject/RequireRW are enforced centrally -- per-call-site rules drift (Reject reached 4/112 handlers). Only context and task list emit JSON documents; init/new adapt (suppress decoration); everyone else refuses. Converting all 40 read commands to emit JSON is separate, larger scope.
