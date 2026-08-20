---
id: f-legacy-runtime-adapters-need-an-explicit-provenance-migration
kind: note
note_kind: finding
created: 2026-08-19T13:23:20Z
created_by: a-maintainer-1cw4s7
about: "[[t-01M0AEYFXAB22RE9Y2SH9WZZKR]]"
severity: moderate
---
# Legacy runtime adapters need an explicit provenance migration
internal/features/execution/preflight.go preserves execution for adapters with no context_provenance while warning that hermeticity is unknown; newly created shipped presets carry all six classes and strict enforcement. Refusing every legacy adapter immediately broke existing fixture and user-defined runtime flows before they could be migrated.
