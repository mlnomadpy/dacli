---
id: f-global-json-flag-is-honored-by-only-4-of-117-commands-40-read-commands-silently
kind: note
note_kind: finding
created: 2026-08-04T18:18:12Z
created_by: a-go-auditor-2ednq4
about: "[[t-01KZ6SAHPQ9ZB2XNTMWC3HPCV5]]"
source_event: 01KZ6SQ3XJCRC5NBH92RXAM0N6
---
# global --json flag is honored by only 4 of ~117 commands; ~40 read commands silently ignore it (exit 0, human text)
CONTRACT: cli.go:159 prints 'Usage: dacli <command> [args] [--json]' — the JSON flag is presented as a universal option. cli.go:105-112 strips it from argv for EVERY command and sets ctx.JSON, so no command ever errors on it.

REALITY: ctx.JSON is read in only 4 non-test slice files that actually EMIT machine output — briefing/briefing.go:20 (context), onboard/new.go:657 (new), insight/overview.go:27 (overview), planning/planning.go:201 (task list). A 5th, wscore/wscore.go:78 (init), only SUPPRESSES getting-started text when JSON is set; it emits no JSON.

EVERY other read/list/show command ignores the flag and prints human text with exit 0. Verified to contain no ctx.JSON reference: status, next, estimate, critical-path, wbs, burndown, velocity, calibrate, taint, doctor, standup, lint (all in insight.go — grep shows ctx.JSON only in that package's overview.go); runs list, runs show, agents, logs (execution.go); team, team route, agent tree, agent show, role list/show (teamops.go); project list/show, task show, risk list, glossary (planning.go); queue list (queues.go); worktree list, pr status (lifecycle.go); loop status (orchestration.go); threads (collab.go); skill list/show, template list/show, runtime list, prompt list/show, blame, contrib, catalog, stage, whoami, replay.

FAILURE SCENARIO: an agent runs 'dacli status --json' (or via MCP — same executor passes jsonMode through cli.go:190-197) expecting a parseable object, receives ASCII-table human text with EXIT 0, and crashes its parser or silently misreads. Because exit is 0 and the flag was silently stripped rather than rejected, nothing signals that JSON mode did nothing.

EVIDENCE: grep -rn 'ctx.JSON' internal/features/*/*.go (excluding _test) returns exactly the 5 files named above.
