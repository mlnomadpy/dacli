---
id: t-01M0ZCAQ33YAXPS79D8EJ676KP
kind: task
created: 2026-08-26T15:51:56Z
created_by: a-root
owner: a-root
github:
  issue: 790
  repo: mlnomadpy/dacli
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
---
# [agent-report] pr defaults to main instead of the linked repository default branch
## Context
Adopted from GitHub issue #790.

When a linked repository uses master as its GitHub default branch, dacli pr --task <ref> omitted --base and invoked PR creation against main, producing GraphQL errors: no commits between main and the task branch / base ref must be a branch. Re-running the same command with --base master succeeded. Expected: resolve the linked repository default branch (or effective project landing base) before PR creation.

---
_Reported via `dacli report`._
- dacli: dev
- platform: darwin/arm64
- workspace and run transcript withheld (public upstream) — re-run with --disclose to include them

## Acceptance
- [x] `dacli pr --task <ref>` resolves the explicit CLI base first, then configured landing base, then the linked repository default branch.
- [x] A linked repository whose default branch is `master` creates the PR against `master` when no base override is configured.
- [x] An explicit or configured non-default landing base is preserved and reported in dry-run and real execution.
- [x] Failure to resolve an authoritative base fails closed before invoking PR creation and names the recovery action.
- [x] Public-command tests cover default-branch discovery, configured override precedence, and remote failure.
- [x] Mutation evidence and the full repository verification gates pass.
## Log
- 2026-08-26T23:22:00Z claimed by a-claude-fixer-6jmk82
- 2026-08-26T23:45:51Z claimed by a-root
- 2026-08-27T00:06:03Z accepted by a-root
- 2026-08-27T00:06:03Z verified by `test -z "$(gofmt -l .)" && GOCACHE=/private/tmp/dacli-go-cache-main-505 go vet ./... && GOCACHE=/private/tmp/dacli-go-cache-main-505 GOLANGCI_LINT_CACHE=/private/tmp/dacli-golangci-cache-main-505 /Users/tahabsn/go/bin/golangci-lint run && GOCACHE=/private/tmp/dacli-go-cache-main-505 go test ./...` (exit 0) in branch main at a4b8733 — proves that tree builds, not that the work is in trunk
- 2026-08-27T00:06:03Z deliverable: dacli/505-agent-report-pr-defaults-to-main-instead-of-the-linked-repository-default-branch is merged into main
- 2026-08-27T00:06:03Z completed by a-root
- 2026-08-27T00:07:20Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/805 (event 01M10845EVM7Y7R9W89QFCSPCT)
## Verification Evidence
{"command":"test -z \"$(gofmt -l .)\" \u0026\u0026 GOCACHE=/private/tmp/dacli-go-cache-505-verify go vet ./... \u0026\u0026 GOCACHE=/private/tmp/dacli-go-cache-505-verify GOLANGCI_LINT_CACHE=/private/tmp/dacli-golangci-cache-505-verify /Users/tahabsn/go/bin/golangci-lint run \u0026\u0026 GOCACHE=/private/tmp/dacli-go-cache-505-verify go test ./...","exit_code":0,"duration_ms":114899,"artifact_hash":"sha256:7296c6ad1ba55e28d7d0eb44bc36867de2b0f1f8b2c2e527a31f608f9c5e6e80","verifier":"a-root","branch":"dacli/505-agent-report-pr-defaults-to-main-instead-of-the-linked-repository-default-branch","commit_sha":"f746f11d7d2c92178f4b509ea7f459897b870b04"}
{"command":"test -z \"$(gofmt -l .)\" \u0026\u0026 GOCACHE=/private/tmp/dacli-go-cache-main-505 go vet ./... \u0026\u0026 GOCACHE=/private/tmp/dacli-go-cache-main-505 GOLANGCI_LINT_CACHE=/private/tmp/dacli-golangci-cache-main-505 /Users/tahabsn/go/bin/golangci-lint run \u0026\u0026 GOCACHE=/private/tmp/dacli-go-cache-main-505 go test ./...","exit_code":0,"duration_ms":66919,"artifact_hash":"sha256:5318f68b7af61af9b2b68c1a24892648f6dd82905a2a357b14a0e8afb8e88e0d","verifier":"a-root","branch":"main","commit_sha":"a4b873397406bb0f9c25c5603eafa59dd0c54391"}
