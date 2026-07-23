---
id: f-docs-walkthrough-md-9-docs-roster-md-now-cover-dacli-loop-the-governor-the
kind: note
note_kind: finding
created: 2026-07-23T12:38:22Z
created_by: a-h0v6gsbgbv
about: [[098]]
severity: minor
---
# docs/WALKTHROUGH.md § 9 + docs/ROSTER.md now cover dacli loop, the governor, the sprint model, and the integrator/auto-merge landing mechanism; README's loop section links to both (docs/WALKTHROUGH.md#9-zooming-out-the-perpetual-loop, docs/ROSTER.md)
Added WALKTHROUGH.md § 9 (sprint-phase table, governor decision table, landing mechanisms). Regenerated docs/ROSTER.md via internal/features/catalog's collectRoles/renderCatalog (workspace pointed directly at this worktree root to bypass workspace.Find's linked-worktree redirect to the main checkout, which would otherwise read stale role files) — the integrator role was previously missing a summary field and absent from the last-generated table; added .dacli/roles/integrator.md:summary so its ROSTER row is meaningful. Also indexed ROSTER.md in docs/README.md, which omitted it entirely. Verified: go build ./... and go test ./internal/features/catalog/... ./internal/features/orchestration/... green; gofmt -l . clean.
