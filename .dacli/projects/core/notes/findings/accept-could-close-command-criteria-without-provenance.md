---
id: f-accept-could-close-command-criteria-without-provenance
kind: note
note_kind: finding
created: 2026-08-13T21:23:18Z
created_by: a-fixer-vqsmp1
about: "[[432]]"
severity: major
---
# Accept could close command criteria without provenance
internal/features/acceptance/acceptance.go accepted a task with a backticked command criterion when --verify was absent; mutation of the new guard makes TestAcceptRefusesCommandCriterionWithoutProvenance fail with accept exit 0.
