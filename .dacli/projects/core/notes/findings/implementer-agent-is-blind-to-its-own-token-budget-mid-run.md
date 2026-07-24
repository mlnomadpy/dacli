---
id: f-implementer-agent-is-blind-to-its-own-token-budget-mid-run
kind: note
note_kind: finding
created: 2026-07-24T09:36:16Z
created_by: a-jtygckd7t5
about: [[140]]
severity: moderate
---
# Implementer agent is blind to its own token budget mid-run
Governor tracks windowSpent (internal/features/orchestration/governor.go:60,69) and --max-tokens refuses over-band spawns at launch (execution.go:294-312), but mid-run the agent has no spent-vs-band gauge against its calibrated band (internal/store/calibration.go:22-28). It learns it overspent only after the fact, too late to trim scope. Reframe of hypothesis H7 that actually reaches the agent.
