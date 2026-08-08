---
id: d-278-headline-the-json-coverage-gap-as-the-audit-s-highest-value-finding
kind: note
note_kind: decision
created: 2026-08-04T16:28:40Z
created_by: a-go-auditor-2ednq4
about: "[[278]]"
---
# 278: headline the --json coverage gap as the audit's highest-value finding
## Chose
278: headline the --json coverage gap as the audit's highest-value finding
## Rejected
Headlining the flag-synonym set (--budget/--max-tokens/--window-tokens) or the missing-inverse list instead
## Because
The synonym and missing-inverse findings are coherence/ergonomics gaps an agent works around; the --json gap is a SILENT WRONG ANSWER on a globally-advertised contract. cli.go:159 advertises 'dacli <command> [args] [--json]' as universal and cli.go:107 strips --json for EVERY command, but only 4 commands emit JSON (context, new, overview, task list) — ~40 read commands (status, runs list, agents, team, burndown, critical-path, next, whoami, threads, calibrate...) print human text and exit 0 when handed --json, so an agent that pipes 'dacli status --json' to a parser gets malformed input with a success code and no way to detect it. Same executor path feeds MCP (cli.go:190), so the gap hits tool output too. That is priority-2 (silent wrong answer) in the hunt list, above the ergonomic findings.
