---
id: f-loop-erased-explicit-implementer-provenance-before-per-task-routing
kind: note
note_kind: finding
created: 2026-08-12T19:24:06Z
created_by: a-codex-maintainer-hyzqzv
about: "[[373]]"
severity: major
---
# Loop erased explicit implementer provenance before per-task routing
internal/features/orchestration/orchestration.go resolved --impl-role and the project default into the same loopCfg.implRole value, then runCycle unconditionally called team.CheapestCapableFor. The red regression TestLoopExplicitImplementerRoleOverridesAutomaticCostRouting observed backend-engineer replaced by frontend-engineer for an estimated task naming docs/RUNTIMES.md.
