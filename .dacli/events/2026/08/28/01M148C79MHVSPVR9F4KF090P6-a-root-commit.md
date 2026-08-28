---
id: 01M148C79MHVSPVR9F4KF090P6
kind: event
schema_version: 1
event_kind: commit
created: 2026-08-28T13:19:03Z
created_by: a-root
about: "[[t-01M146BA62817V08T9P6D6REKT]]"
origin: agent
applied: true
checksum: sha256:571ff2b5f6a2be5eead06e86c7e964bd0f96af28dc7f1882b304631253c6f0a6
---
bc91d118 542: harden reconciliation against false positives

Restrict GitHub findings to nonterminal tasks, distinguish non-task mailbox events, remove the invalid blocked-without-dependency inference, match the loop marker's .dacli-excluding semantics, and bound GitHub observation time.

Mutation proof: removing the completed-task filter made TestReconcileOmitsHistoricalCompletedDelivery fail with: completed historical delivery leaked into current findings.
role: root
