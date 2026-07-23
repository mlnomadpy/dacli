---
id: t-01KY849P3TJ9GBW069H666DVEX
kind: task
created: 2026-07-23T18:37:38Z
created_by: a-root
owner: a-root
github:
  issue: 76
  repo: mlnomadpy/dacli
---
# runtime presets should default to usage_format: stream-json (else --tail + calibration are silently blind)
## Context
Adopted from GitHub issue #76.

## What I hit

The `cc` / `cc-rw` runtimes shipped **without** `usage_format: stream-json`. Two consequences bit me during a live run:

1. **`dacli agents --tail` was blind.** Plain `claude --print` buffers all stdout and flushes only at exit, so a non-detached spawn's `transcript.log` stayed 0 bytes for the entire run — `--tail` showed `(no transcript output yet)` until the agent finished. No thinking-vs-hung signal for minutes.
2. **Calibration captured nothing.** No usage actuals were recorded, so the `--window-tokens` budget governor was a no-op and `calibrate` had no token bands.

Enabling `usage_format: stream-json` on both runtimes fixed *both* at once: transcripts stream live (tool markers + reasoning), and every spawn feeds calibration.

## Impact

The two headline observability/governance features (`agents --tail`, token-budget governor + `calibrate`) are silently inert on a fresh workspace until someone knows to flip this flag. It's a poor default for a tool whose whole pitch is visibility into an agent fleet.

## Suggested direction

- Make `runtime add --preset claude-code` set `usage_format: stream-json` **by default** (the claude CLI supports `--print --output-format stream-json --verbose`).
- `runtime doctor` could warn when a claude-family runtime has no `usage_format` ("`--tail` and calibration will be blind — enable stream-json").
- Consider the same default for `--preset generic-exec` where the adapter supports a streaming JSON mode.

## Acceptance
## Log
