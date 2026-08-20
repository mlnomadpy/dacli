---
id: t-01M0D2KPCZ5PEFXJS4B0J59Z5C
kind: task
created: 2026-08-19T13:15:45Z
created_by: a-root
owner: a-root
github:
  issue: 727
  repo: mlnomadpy/dacli
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
---
# Fix the truncated dacli commit help contract and remove its parity exception
## Context
Adopted from GitHub issue #727.

## Reproduction

On current main after PR #719:

```text
$ dacli commit --help
dacli commit

Commit as yourself: author = agent (role), with dacli trailers

dacli commit \\
```

The handler rejects missing arguments with the real contract:

```text
dacli commit "<message>" [--task ref] [--no-add] [--force]
```

Source currently declares `Usage: "dacli commit \\\\"` in `internal/features/vcs/vcs.go`, while `internal/cli/usage_parity_invariant_test.go` carries the correct form as an override. This lets the global usage-parity invariant pass while user-facing help remains malformed.

## Acceptance
- [x] `dacli commit --help` prints `dacli commit "<message>" [--task ref] [--no-add] [--force]`.
- [x] The command table and handler missing-argument error share the same canonical contract without a commit-specific parity exception.
- [x] A focused CLI/VCS test covers both help and missing-argument output.
- [x] Mutation evidence restores the truncated literal and makes the invariant or focused test fail.
- [x] `go test ./...` and the repository verification bar pass.
## Log
- 2026-08-19T14:07:30Z claimed by a-fixer-gcha7z
- 2026-08-19T14:23:32Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/744 (event 01M0D6CBXKBEJAQQ0YF3780JJN)
- 2026-08-19T14:30:27Z accepted by a-root
- 2026-08-19T14:30:27Z verified by `GOCACHE=/tmp/dacli-accept-483 go test ./internal/features/vcs ./internal/cli` (exit 0) in branch main at a0ed6bc — proves that tree builds, not that the work is in trunk
- 2026-08-19T14:30:27Z deliverable: dacli/483-fix-the-truncated-dacli-commit-help-contract-and-remove-its-parity-exception is merged into main
- 2026-08-19T14:30:27Z completed by a-root
## Verification Evidence
{"command":"GOCACHE=/tmp/dacli-accept-483 go test ./internal/features/vcs ./internal/cli","exit_code":0,"duration_ms":2732,"artifact_hash":"sha256:d6489d23b62539c3c9d0296cfdc836255eb9bf7644bbe2922ee20ee458f0b2ed","verifier":"a-root","branch":"main","commit_sha":"a0ed6bcd92d6d3eea5cc1b00fcce110e9adf29a9"}
{"command":"GOCACHE=/tmp/dacli-accept-483 go test ./internal/features/vcs ./internal/cli","exit_code":0,"duration_ms":2018,"artifact_hash":"sha256:d6489d23b62539c3c9d0296cfdc836255eb9bf7644bbe2922ee20ee458f0b2ed","verifier":"a-root","branch":"main","commit_sha":"a0ed6bcd92d6d3eea5cc1b00fcce110e9adf29a9"}
