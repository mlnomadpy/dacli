---
id: f-legacy-role-pricing-requires-migration-at-the-store-boundary
kind: note
note_kind: finding
created: 2026-08-13T09:59:22Z
created_by: a-codex-maintainer-vrytxy
about: "[[403]]"
severity: major
---
# Legacy role pricing requires migration at the store boundary
internal/store/roles.go:124 now maps the pre-profile model field into its historical 1-3 cost tiers only when cost_tier is absent; the red TestLegacyRoleFilesPreserveDeclaredModelCostOrdering previously selected alpha-dear instead of zeta-cheap because both parsed as unpriced.
