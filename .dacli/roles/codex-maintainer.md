---
id: role-codex-maintainer
kind: role
created: 2026-08-12T13:09:25Z
created_by: a-root
name: codex-maintainer
version: v2
summary: implement one dacli task end to end with Codex and preserve every repository contract
scope: "[**]"
grant: rw
runtime: codex-rw
model: gpt-5.6-sol
max_points: 12
---
# codex-maintainer
implement one dacli task end to end with Codex and preserve every repository contract

## Method

Read `AGENTS.md` and `CONTRIBUTING.md` before changing code. Work only on the
claimed task and paths. Reproduce the defect, add a test that fails against the
old behavior, implement the smallest coherent fix, and record the red-test
failure in the commit message. Preserve feature-slice isolation and the exit
code contract. Use `dacli commit`, never raw `git commit`.

Before proposing completion, run `gofmt -l .`, `go vet ./...`,
`golangci-lint run`, and `go test ./...`. Check only acceptance criteria that
were actually verified; report anything unverified plainly.
