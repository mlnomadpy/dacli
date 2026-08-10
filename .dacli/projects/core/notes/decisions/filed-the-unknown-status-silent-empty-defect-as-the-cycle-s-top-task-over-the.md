---
id: d-filed-the-unknown-status-silent-empty-defect-as-the-cycle-s-top-task-over-the
kind: note
note_kind: decision
created: 2026-08-10T15:19:04Z
created_by: a-go-auditor-qz3zb9
about: "[[303]]"
---
# Filed the unknown --status silent-empty defect as the cycle's top task, over the CheckAllAcceptance section-rewrite data loss
## Chose
Filed the unknown --status silent-empty defect as the cycle's top task, over the CheckAllAcceptance section-rewrite data loss
## Rejected
Filing the CheckAllAcceptance Acceptance-section data loss (store.go:301-315 replaces the section with only flattened checkbox lines, dropping prose/nested items on every close) instead
## Because
Both are code-proven. But the --status bug is LIVE and reachable today on read commands an operator/loop runs constantly (task list, task list --json, lint), where an ordinary typo yields exit-0 empty that reads as 'backlog empty' — a record disagreeing with reality with no precondition beyond a mistyped value. The CheckAllAcceptance data loss is real but LATENT: it only destroys content when an Acceptance section contains non-checkbox lines or nested/indented checkboxes, and a scan of every core task file found zero such instances today, so nothing is currently lost. I recorded CheckAllAcceptance separately as a finding for a later cycle; the --status refusal is the smaller, higher-reachability fix and is the better single pick.
