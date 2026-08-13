---
id: 01KZXF30TZJEA4SETKZ9THEYVZ
kind: event
event_kind: commit
created: 2026-08-13T11:45:59Z
created_by: a-root
about: "[[t-01KZXET2V7JWH2AP2XARJTJPVT]]"
origin: agent
applied: true
---
ab3b764 411: hydrate verified runtime probes consistently

Load persisted runtime-doctor RO evidence through one store boundary and use it
from verify, preflight, and spawn. This prevents grant enforcement from
silently falling back to declaration-only unknown state.

Red: TestVerifyLoadsPersistedRuntimeROProbe reproduced verify's sandbox probe:
unknown refusal before the shared hydrated loader.
role: root
