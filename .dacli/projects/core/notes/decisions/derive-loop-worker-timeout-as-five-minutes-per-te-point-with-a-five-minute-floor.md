---
id: d-derive-loop-worker-timeout-as-five-minutes-per-te-point-with-a-five-minute-floor
kind: note
note_kind: decision
created: 2026-08-12T15:38:24Z
created_by: a-codex-maintainer-vxzmpg
about: "[[378]]"
github:
  issue: 495
  repo: mlnomadpy/dacli
---
# Derive loop worker timeout as five minutes per Te point with a five-minute floor
## Chose
Derive loop worker timeout as five minutes per Te point with a five-minute floor
## Rejected
Keep a fixed default or derive only from role capacity
## Because
The task estimate describes the actual worker's scope, already exists on both implementation tasks and the standing review anchor, and scales larger tasks beyond the historical 300-second kill while preserving 300 seconds for unsized/sub-point work; --worker-timeout explicitly overrides policy.
