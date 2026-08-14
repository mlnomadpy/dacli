---
id: f-loop-landing-mode-was-inferred-from-origin-instead-of-project-policy
kind: note
note_kind: finding
created: 2026-08-14T01:04:09Z
created_by: a-maintainer-0c0w8g
about: "[[448]]"
severity: major
---
# Loop landing mode was inferred from origin instead of project policy
internal/features/orchestration/orchestration.go previously selected PR landing from origin presence via prMode, bypassing model.ResolveLanding and silently choosing local when configured PR mode had no remote. The new resolver consumes project policy and returns exit 3 before spawning when effective PR mode cannot reach origin.
