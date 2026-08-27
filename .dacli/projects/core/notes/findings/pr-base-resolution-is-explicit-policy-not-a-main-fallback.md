---
id: f-pr-base-resolution-is-explicit-policy-not-a-main-fallback
kind: note
note_kind: finding
created: 2026-08-26T23:54:36Z
created_by: a-root
about: "[[505]]"
severity: major
---
# PR base resolution is explicit policy not a main fallback
Task 505 makes dacli pr resolve explicit --base, then the task project's configured landing base, then gh repo view's linked repository default. It reports the branch/source in dry-run and real paths and fails closed before pr create with configuration recovery. Mutation forcing main makes TestPRUsesLinkedRepositoryDefaultMaster fail because pr create receives --base main. Restored full gates pass.
