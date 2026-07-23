---
id: t-01KY849P45GTAKNHWRRAWVSZSJ
kind: task
created: 2026-07-23T18:37:38Z
created_by: a-root
owner: a-root
github:
  issue: 72
  repo: mlnomadpy/dacli
---
# loop: governor persistence collides with --max-cycles (fresh bounded runs halt immediately)
## Context
Adopted from GitHub issue #72.

## What I hit

Driving a build-itself sprint program, I ran `dacli loop --project core --width 1 --max-cycles 1` once per cycle. It worked for the first several cycles. Then **every subsequent `--max-cycles 1` invocation halted instantly** with `● halt: reached --max-cycles 1`, building nothing.

## Root cause

Governor state persistence (the `core-governor.txt` snapshot) restores the **cumulative** cycle counter across process invocations. `Governor.Before` then evaluates `cycle >= MaxCycles` against that restored total — so once the persisted count reaches N, a fresh `--max-cycles N` run sees e.g. `9 >= 1` and halts before doing any work.

Persistence is correct for *resuming a killed `--yolo` run*, but it silently breaks *repeated bounded invocations*, which is the natural way to drive the loop one cycle at a time (e.g. when the unattended `--yolo` path is gated by a permission policy).

## Impact

The single-cycle driving pattern — arguably the safest way to run the loop under supervision — becomes unusable after the persisted count crosses the bound, with a halt message that misattributes the cause.

## Suggested direction

`--max-cycles` should bound cycles run **in the current invocation**, not the cumulative persisted total. Options:
- Track an invocation-local cycle counter for the `--max-cycles` gate, keep the cumulative count only for reporting/resume; or
- On a fresh (non-resumed) invocation, reset the bounded counter; or
- Add `--resume`/`--fresh` to make the intent explicit.

## Workaround

Drop `--max-cycles` entirely and rely on the non-`--yolo` checkpoint return (`dacli loop --project core --width 1` runs exactly one cycle and stops), which is unaffected by the persisted count.

## Acceptance
## Log
