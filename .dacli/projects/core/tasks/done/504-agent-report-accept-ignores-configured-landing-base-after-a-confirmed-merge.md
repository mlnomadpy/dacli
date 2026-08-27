---
id: t-01M0ZCAQ05J2H9VHB4BA9YTQGD
kind: task
created: 2026-08-26T15:51:56Z
created_by: a-root
owner: a-root
github:
  issue: 791
  repo: mlnomadpy/dacli
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
---
# [agent-report] accept ignores configured landing base after a confirmed merge
## Context
Adopted from GitHub issue #791.

Observed with landing.mode=pr and landing.base=dev: after the task pull request was confirmed merged into dev, dacli accept still evaluated the task commit against master and refused closure as unlanded. Manual workaround: verify the merged PR and fresh dev ancestry, then use --allow-unlanded. Related but distinct from #790, which covers PR creation choosing a default base. Expected: acceptance resolves the effective landing base and authoritative merged-PR state, so a PR merged into configured dev is accepted without an override. Acceptance criteria: add a regression with repository default master, configured landing base dev, and a confirmed merge into dev; prove accept succeeds and never treats master as the target. Non-goal: changing the repository default branch.

---
_Reported via `dacli report`._
- dacli: dev
- platform: darwin/arm64
- workspace and run transcript withheld (public upstream) — re-run with --disclose to include them

## Acceptance
- [x] Acceptance resolves the project’s effective landing base rather than assuming `main` or the repository default branch.
- [x] A regression with repository default `master`, configured landing base `dev`, and a confirmed PR merge into `dev` accepts without `--allow-unlanded`.
- [x] The same regression proves acceptance never evaluates the task commit against `master` when `dev` is effective.
- [x] Remote lookup or ancestry failures fail closed with the selected base named in the diagnostic.
- [x] Mutation evidence and the full repository verification gates pass.
## Log
- 2026-08-26T23:13:39Z claimed by a-root
- 2026-08-26T23:42:38Z accepted by a-root
- 2026-08-26T23:42:38Z verified by `test -z "$(gofmt -l .)" && GOCACHE=/private/tmp/dacli-go-cache-main-504 go vet ./... && GOCACHE=/private/tmp/dacli-go-cache-main-504 GOLANGCI_LINT_CACHE=/private/tmp/dacli-golangci-cache-main-504 /Users/tahabsn/go/bin/golangci-lint run && GOCACHE=/private/tmp/dacli-go-cache-main-504 go test ./...` (exit 0) in branch main at f746f11 — proves that tree builds, not that the work is in trunk
- 2026-08-26T23:42:38Z deliverable: dacli/504-agent-report-accept-ignores-configured-landing-base-after-a-confirmed-merge is merged into main
- 2026-08-26T23:42:38Z completed by a-root
- 2026-08-26T23:43:56Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/804 (event 01M106RZGR9SVHA203JJ8AES7A)
## Verification Evidence
{"command":"test -z \"$(gofmt -l .)\" \u0026\u0026 GOCACHE=/private/tmp/dacli-go-cache go vet ./... \u0026\u0026 GOCACHE=/private/tmp/dacli-go-cache GOLANGCI_LINT_CACHE=/private/tmp/dacli-golangci-cache-504-verify /Users/tahabsn/go/bin/golangci-lint run \u0026\u0026 GOCACHE=/private/tmp/dacli-go-cache go test ./...","exit_code":0,"duration_ms":6588,"artifact_hash":"sha256:8771343e611731ea2bb4c5b02ee67d0f490ce61629118bea437c7011418354c0","verifier":"a-root","branch":"dacli/504-agent-report-accept-ignores-configured-landing-base-after-a-confirmed-merge","commit_sha":"d909be6ce6828db9da41d921c9839d589cf4d466"}
{"command":"test -z \"$(gofmt -l .)\" \u0026\u0026 GOCACHE=/private/tmp/dacli-go-cache-main-504 go vet ./... \u0026\u0026 GOCACHE=/private/tmp/dacli-go-cache-main-504 GOLANGCI_LINT_CACHE=/private/tmp/dacli-golangci-cache-main-504 /Users/tahabsn/go/bin/golangci-lint run \u0026\u0026 GOCACHE=/private/tmp/dacli-go-cache-main-504 go test ./...","exit_code":0,"duration_ms":89129,"artifact_hash":"sha256:e316f65603e461e0a9dfaf95f57415505d794bfa837ced14830db089c75c46ae","verifier":"a-root","branch":"main","commit_sha":"f746f11d7d2c92178f4b509ea7f459897b870b04"}
