---
id: f-task-408-claim-excludes-its-required-contracts-directory
kind: note
note_kind: finding
created: 2026-08-13T14:26:42Z
created_by: a-fixer-6nxvsp
about: "[[408]]"
severity: major
---
# Task 408 claim excludes its required contracts directory
dacli commit refused all 16 implementation files because the live claim is [internal/features/execution], but acceptance criterion 1 explicitly requires contracts/controlplane/v1. The completed verified changes are entirely under contracts/controlplane/v1. Per claim isolation I did not use --force or edit internal/features/execution. Owner must correct the claim to contracts/controlplane/v1 and recommit this worktree; the refusal is final and was not retried.
