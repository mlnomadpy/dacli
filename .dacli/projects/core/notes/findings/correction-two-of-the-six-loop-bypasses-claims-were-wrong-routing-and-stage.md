---
id: f-correction-two-of-the-six-loop-bypasses-claims-were-wrong-routing-and-stage
kind: note
note_kind: finding
created: 2026-08-10T10:22:25Z
created_by: a-root
about: "[[300]]"
severity: major
origin: internal/features/orchestration/orchestration.go:645,489
---
# Correction: two of the six 'loop bypasses' claims were wrong — routing and stage gates are wired
Re-read the code before acting on my own finding, and two of its six claims do not hold. (1) team assign: capacity routing IS wired — orchestration.go:645-677 loads the roster, derives the fallback role's kind, and routes each task through team.CheapestCapable on its own Te (landed as dacli 233). My claim that implRole is fixed per wave described the pre-233 behavior. (2) stage gates: advanceStages() is called every cycle at orchestration.go:489. Neither is a bypass. What IS real, and is the thread both wrong claims were groping at: routing and critical-path ordering BOTH silently degrade when a task has no estimate (:672 skips CheapestCapable, :1761 drops haveCPM), and the loop never sizes one. Also real: lint is never run, so a vague acceptance criterion filed by the review phase goes straight to an implementer. Those two are the fix; the other four claims should not be actioned.
