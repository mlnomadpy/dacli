---
id: f-task-433-implementation-is-staged-but-worktree-ownership-blocks-commit
kind: note
note_kind: finding
created: 2026-08-13T21:27:16Z
created_by: a-fixer-q7facv
about: "[[433]]"
severity: major
---
# Task 433 implementation is staged but worktree ownership blocks commit
In .dacli/worktrees/core-433-export-scenario-metrics-through-a-stable-json-interface, full go test ./..., go vet ./..., gofmt -l ., focused metrics/insight/wscore/cli/execution tests, and the failure-class mutation proof passed. golangci-lint was unavailable. dacli commit refused: worktree owned by a-fixer-fwr9f3 as a-fixer-q7facv; staged work was preserved. Exit 3 was not retried.
