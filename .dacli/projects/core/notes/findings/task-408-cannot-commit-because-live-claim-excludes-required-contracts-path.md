---
id: f-task-408-cannot-commit-because-live-claim-excludes-required-contracts-path
kind: note
note_kind: finding
created: 2026-08-13T14:36:11Z
created_by: a-fixer-z4s0m6
about: "[[408]]"
severity: major
---
# Task 408 cannot commit because live claim excludes required contracts path
The verified implementation is staged in contracts/controlplane/v1 (16 files), but dacli commit returned exit 3 because this run claims only [internal/features/execution]. Acceptance criterion 1 explicitly requires contracts/controlplane/v1. Per claim isolation, --force was not used and the refusal was not retried. Owner must correct the claim to contracts/controlplane/v1, then commit, push, and open the PR from branch dacli/408-define-the-signed-control-plane-event-protocol.
