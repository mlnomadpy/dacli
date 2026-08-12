---
id: f-task-364-implementation-committed-on-isolated-branch
kind: note
note_kind: finding
created: 2026-08-12T16:23:54Z
created_by: a-codex-maintainer-xm4nzv
about: "[[364]]"
severity: major
---
# Task 364 implementation committed on isolated branch
Commit dc70055 on branch dacli/364-fix-cross-process-lost-updates-in-direct-task-mutations adds the deterministic real-process [2/2] regression and wraps direct task RMW writers in store.WithTask. Red proof: taskcheck_concurrency_test.go:56 persisted acceptance = [1/2], want [2/2]. Affected suites, gofmt, go vet pass; full race remains unverified because sandbox process-observation tests fail, and golangci-lint is unavailable.
