---
id: 01KZ6SQG9E120445RZQW9B7ZV4
kind: event
event_kind: finding
created: 2026-08-04T16:29:24Z
created_by: a-go-auditor-2ednq4
about: "[[t-01KZ6SAHPQ9ZB2XNTMWC3HPCV5]]"
origin: agent
applied: true
---
one token-ceiling concept lives under 4 flag names, and --budget/--all are homonyms across commands

SAME MEANING, DIFFERENT NAMES (a token/cost ceiling):
- spawn --budget (execution.go:386, recorded soft budget) AND spawn --max-tokens (execution.go:502, HARD pre-launch gate that refuses exit 3) — TWO token knobs on ONE command; names give no hint which enforces.
- verify --budget (execution/verify.go:73)
- loop --max-tokens (orchestration.go:132) AND loop --window-tokens / --budget-window (orchestration.go:148,149,163)
So the same 'cap on tokens' idea is spelled --budget, --max-tokens, --window-tokens, and --budget-window across spawn/verify/loop. An agent that learned 'spawn --budget' cannot predict that the loop's equivalent is --max-tokens/--window-tokens.

SAME NAME, DIFFERENT MEANINGS (homonyms):
- --budget: brief-token size cap for the context brief (briefing/briefing.go:34,76) vs a recorded plan budget on spawn (execution.go:386) vs verify's token allotment (verify.go:73). Three unrelated referents.
- --all: 'all acceptance boxes of ONE task' in task check (planning.go:360) vs 'every proposed task' in accept (acceptance.go:64) vs 'all live agents' in kill (execution.go:1917). Three different scopes of 'all'.

CONSEQUENCE: the task's premise — 'an agent can predict a command it has not used from the ones it has' — fails here: transferring --budget or --all from one command to another silently does the wrong thing (e.g. 'kill --all' vs an agent expecting per-task semantics).

EVIDENCE: file:line above; grep -rnoE 'f\.(Get|Bool|All|Int)\("(budget|max-tokens|window-tokens|budget-window|all)"' internal/features/.
