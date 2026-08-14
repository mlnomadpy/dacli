---
id: t-01KZYA0JT4Q7PM1F1PEHRFFVSF
kind: task
created: 2026-08-13T19:36:31Z
created_by: a-codex-loop-auditor-hxqjcg
owner: a-root
priority: should
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
parent: "[[t-01KZXXZPBXT332W00RP94HTR2K]]"
github:
  issue: 607
  repo: mlnomadpy/dacli
---
# Export scenario metrics through a stable JSON interface
## So that
repeatable runs can be compared by tooling rather than scraping human text
## Acceptance
- [x] insight and wscore consume one internal metrics interface for completion, retry, failure class, wall time, token budget, and human intervention
- [x] dacli metrics --json emits a documented stable object containing every metric and its sample counts
- [x] tests compare two named scenario windows and reject missing or fabricated token and failure-class data
## Log
- 2026-08-13T21:18:02Z claimed by a-fixer-fwr9f3
- 2026-08-13T21:53:10Z adopted by a-root (owner a-codex-loop-auditor-hxqjcg orphaned)
- 2026-08-13T21:53:10Z accepted by a-root (applied 1 proposal(s))
- 2026-08-13T21:53:10Z verified by `go test ./internal/metrics ./internal/features/insight ./internal/features/wscore ./internal/cli ./internal/features/execution` (exit 0) in branch main at f066440 — proves that tree builds, not that the work is in trunk
- 2026-08-13T21:53:10Z deliverable: dacli/433-export-scenario-metrics-through-a-stable-json-interface is merged into main
- 2026-08-13T21:53:10Z completed by a-root
- 2026-08-13T23:51:31Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/627 (event 01KZYGMT4TAW3R49PR6T80SMBR)
## Verification Evidence
{"command":"go test ./internal/metrics ./internal/features/insight ./internal/features/wscore ./internal/cli ./internal/features/execution","exit_code":0,"duration_ms":41503,"artifact_hash":"sha256:d0351b67c584e7cecb522ac2129bdc494780487ee58bdbc3156879667b54dfd8","verifier":"a-root","branch":"main","commit_sha":"f066440e09812861d8ad4ab89cb135e793a4bf9a"}
