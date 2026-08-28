---
id: f-owner-must-materialize-verified-task-542-acceptance
kind: note
note_kind: finding
created: 2026-08-28T12:56:23Z
created_by: a-maintainer-k0gy77
about: "[[t-01M146BA62817V08T9P6D6REKT]]"
severity: moderate
---
# Owner must materialize verified task 542 acceptance
task check --n 1 returned policy refusal: only owner a-root may check acceptance. Implementation regressions pass in internal/delivery/delivery_test.go and internal/features/insight/insight_test.go; owner must materialize criteria after review.
