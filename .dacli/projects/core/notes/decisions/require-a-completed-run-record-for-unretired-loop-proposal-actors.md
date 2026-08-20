---
id: d-require-a-completed-run-record-for-unretired-loop-proposal-actors
kind: note
note_kind: decision
created: 2026-08-19T11:40:51Z
created_by: a-fixer-4y3mj5
about: "[[t-01M0AKHSFGWWSMDFCWCE9RYCGQ]]"
github:
  issue: 712
  repo: mlnomadpy/dacli
---
# Require a completed run record for unretired loop proposal actors
## Chose
Require a completed run record for unretired loop proposal actors
## Rejected
Treat every known actor without a live run as finished
## Because
A known identity may never have spawned; requiring a valid non-live proc record proves terminal lifecycle while live, unknown, and malformed evidence remain fail-closed
