---
id: t-01KY4ZWW4WCW130QCC202P2Q8G
kind: task
created: 2026-07-22T13:23:01Z
created_by: a-root
owner: a-root
priority: could
estimate: {optimistic: 2, probable: 4, pessimistic: 6}
---
# D4: dogfood-as-proof — the repo own workspace as the reference demo
## Context
D4 captures the differentiator — "attributed log → learn from it" — as a REPRODUCIBLE demo built from this repo's OWN `.dacli/` record. The proof is that the tool developed itself this session (tasks 016–029: resource-monitor, audit fixes, async spawn, shadowing fix, D1–D3), and that history is real self-hosted data sitting in `.dacli/`.

Deliverables (this is a docs + script task — do NOT change Go code):
- **`scripts/dogfood-demo.sh`** — a runnable script (uses the repo-root `./dacli`; build it first if absent with `go build -o ./dacli .`) that prints, with short narration between each:
  1. `./dacli contrib` — per-role/agent contribution rollup (agents authored the code, provably).
  2. `./dacli calibrate` — framed as the empirical Cone of Uncertainty: call out that agent wall-time runs ~0.01–0.03 hours/point, ~50–100× under human-scaled points (the finding that NAMED the D-series).
  3. `./dacli taint external:` — a real blast-radius query over the workspace's own note/event origins (show it end-to-end; if no external origins exist, say so honestly and show the query returning clean).
  4. `./dacli status` and `./dacli standup` — the tree state and per-agent roll-up, derived from the log.
  Keep it read-only (no spawns, no writes). Make it re-runnable and `set -e`-clean.
- **`docs/DOGFOOD.md`** — a short narrative that embeds representative REAL output from running the script against this repo (run it, paste actual output), and explains what each readout proves. Frame calibrate as "dacli measured its own Cone." Be honest where data is thin (e.g. agent-band calibration is still populating).

## Scope (STRICT) — touch ONLY:
- `scripts/` (new)
- `docs/DOGFOOD.md` (new)

## Staging discipline
Do NOT `git add -A`. `git add` ONLY `scripts/dogfood-demo.sh` and `docs/DOGFOOD.md` plus this task's file. `go build ./...` must still pass (you added no Go). `dacli note add finding` with a one-line summary, then `dacli commit`. Box-checking is owner-only — file a completion finding.

## Acceptance
- [x] a reproducible demo script replays dacli's own development from its .dacli record
- [x] the calibrate readout is presented as the empirical Cone of Uncertainty from real self-hosted data
- [x] a real finding's taint blast-radius is shown end to end
- [x] committed on branch by an agent; build + test green
## Log
- 2026-07-22T14:03:11Z completed by a-root
