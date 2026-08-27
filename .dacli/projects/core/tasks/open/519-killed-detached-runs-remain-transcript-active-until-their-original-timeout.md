---
id: t-01M11HZW5KX6JN902621F6F0ZH
kind: task
created: 2026-08-27T12:09:21Z
created_by: a-root
owner: a-root
github:
  issue: 814
  repo: mlnomadpy/dacli
---
# Killed detached runs remain transcript-active until their original timeout
## Context
Adopted from GitHub issue #814.

## Reproduction

1. Spawn a detached run with a non-empty transcript and a long `timeout_s`.
2. Terminate/retire the worker so the recorded PID/PGID no longer exists and `killed.txt` is present.
3. Run `dacli wait <run-id>` or attempt governed worktree reclaim/commit.

Observed on run `01M119NDBGEC4RBBNE5E4MAHM8`: the process tree was gone and `killed.txt` was durable, but `dacli wait` continued reporting the run live until the original 7200-second deadline.

## Root cause

`runLifecycleLive` checks `runtime-exit.txt`, process identity, startup grace, and transcript activity. It does not treat the durable `killed.txt` marker as terminal evidence. For a non-empty transcript and configured timeout, the fallback keeps returning `transcript active` until `Started + Timeout`, even when `dacli kill` has already established that no process remains.

## Impact

- Governed reclaim/commit is refused for the full original timeout.
- `dacli agents` reports a processless retired worker as live/stalled.
- Operators are pressured toward manually editing run artifacts or bypassing dacli.

## Expected behavior

After a governed kill has durably recorded `killed.txt` and process reconciliation confirms the recorded identity is gone, `dacli wait` should finalize the run immediately and release its claims. A regression should cover a non-empty transcript, long configured timeout, missing process, and durable kill marker.

## Acceptance
## Log
