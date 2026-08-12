---
id: 01KZVD6SWKN3ZMCGZ9JG5P6A11
kind: event
event_kind: commit
created: 2026-08-12T16:34:37Z
created_by: a-root
about: "[[t-01KZVCRMZYWGY0P2WA38E344KB]]"
origin: agent
applied: true
---
5929a7f 387: distinguish Codex sandbox startup from write denial

Require a post-attempt command marker before caching read-only verification.

Red proof:
--- FAIL: TestRuntimeDoctorCodexProbeRejectsOuterSandboxStartupFailure
runrecord_test.go:1006: doctor false-verified an outer sandbox startup failure
role: root
