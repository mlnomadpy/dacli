---
id: f-loop-delegated-a-required-landing-transaction-to-the-worker-prompt
kind: note
note_kind: finding
created: 2026-08-27T10:04:04Z
created_by: a-root
about: "[[503]]"
severity: major
---
# Loop delegated a required landing transaction to the worker prompt
Reproduced in orchestration: after branchHasWork, PR mode only appended pending state and assumed the worker had already pushed/opened a PR. The controller now runs task-addressed dacli push then idempotent dacli pr --base <effective> --auto before tracking merge. Mutation: removing queueTaskPR makes TestDriverRunsSprintPhasesInOrder fail because public push/pr phases disappear. Full gofmt, vet, lint, and test gates pass.
