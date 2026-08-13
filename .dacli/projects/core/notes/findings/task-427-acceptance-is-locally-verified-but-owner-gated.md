---
id: f-task-427-acceptance-is-locally-verified-but-owner-gated
kind: note
note_kind: finding
created: 2026-08-13T19:07:50Z
created_by: a-codex-maintainer-mjejj8
about: "[[427]]"
severity: major
---
# task 427 acceptance is locally verified but owner-gated
All three criteria are observed by internal/cli/vcs_test.go: malformed/root invocations preserve the staged claimed file and name both actors, while the child commit has requested subject plus Dacli-Agent/Role/Task trailers. Focused tests and go test ./... pass. task check 427 --n 1/2/3 each returned policy refusal because only owner a-codex-loop-auditor-et4f9e may check boxes. golangci-lint is unavailable in this environment.
