---
id: d-use-a-bounded-startup-grace-plus-recent-transcript-activity-as-shared-run
kind: note
note_kind: decision
created: 2026-08-12T20:18:25Z
created_by: a-codex-maintainer-w6vv23
about: "[[400]]"
github:
  issue: 542
  repo: mlnomadpy/dacli
---
# Use a bounded startup grace plus recent transcript activity as shared run liveness
## Chose
Use a bounded startup grace plus recent transcript activity as shared run liveness
## Rejected
Trust only the recorded guardian PID or add vendor-specific Codex process discovery
## Because
The two observed Codex workers outlived the registered guardian while retaining the transcript descriptor; transcript freshness is durable cross-process evidence, while vendor process matching is brittle. A fixed grace protects registration startup and a fixed freshness window eventually releases genuinely dead launches.
