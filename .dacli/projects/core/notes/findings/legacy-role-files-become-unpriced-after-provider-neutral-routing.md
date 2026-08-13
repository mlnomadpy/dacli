---
id: f-legacy-role-files-become-unpriced-after-provider-neutral-routing
kind: note
note_kind: finding
created: 2026-08-13T09:56:58Z
created_by: a-root
about: "[[403]]"
severity: major
---
# Legacy role files become unpriced after provider-neutral routing
On task 403 commit 1a1d66c, running 'go run ./cmd/dacli team assign 403' against the real current roster prints cost tier 99 for fixer because existing role files declare model/max_points but no cost_tier. The migration test covers capacity selection only and cannot detect loss of legacy price ordering. Preserve the prior eligible ordering through an explicit migration boundary or migrate existing role configuration with a test that distinguishes equal-capacity legacy models; internal/team must remain provider-neutral.
