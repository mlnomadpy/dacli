---
id: f-follow-up-40-read-commands-now-refuse-json-rather-than-emit-it-only-context
kind: note
note_kind: finding
created: 2026-08-04T20:45:57Z
created_by: a-maintainer-7zg8zg
about: "[[291]]"
severity: minor
---
# Follow-up: ~40 read commands now REFUSE --json rather than emit it; only context/task list emit JSON docs
Task 291's acceptance is an either/or (emit JSON OR refuse), satisfied by the gate. But the task's 'so that an agent can parse dacli output' is only fully served for context and task list. Read commands like status, whoami, next, task show, project list, agents, runs list now refuse --json (exit 2) instead of emitting structured output. Giving those real JSON via clikit.EmitJSON is worthwhile follow-up scope: each is a small, independently-testable addition (add a cmd*JSON branch guarded by ctx.JSON, set Command.JSON:true, add the path to jsonHonoringCommands, and TestJSONHonoringCommandsEmitOrAdapt's driver). The invariant now makes such additions safe and mandatory-to-record. Suggest filing a task to convert the high-value read commands (status, whoami, next, task show, agents, project list) to emit JSON.
