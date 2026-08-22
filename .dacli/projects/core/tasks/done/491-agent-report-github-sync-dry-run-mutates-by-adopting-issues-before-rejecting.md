---
id: t-01M0F8DMG5NA6198RGF59NKXWC
kind: task
created: 2026-08-20T09:35:47Z
created_by: a-root
owner: a-root
github:
  issue: 760
  repo: mlnomadpy/dacli
estimate: "{optimistic: 5, probable: 8, pessimistic: 13}"
---
# [agent-report] github sync --dry-run mutates by adopting issues before rejecting the flag
## Context
Adopted from GitHub issue #760.

Running 'dacli github sync bashnota --dry-run' on a linked public repository adopted 12 GitHub issues as tasks, printed 'pull: 12 adopted', and only afterward failed with 'unknown flag(s): --dry-run'. The shipped GitHub landing documentation explicitly recommends this preview command. Expected: dry-run performs no local or remote mutations, or rejects unsupported flags before any operation. Actual: inbound adoption mutated dacli state before validation failure. Please validate flags before starting sync and add a regression that snapshots task/event state around sync --dry-run.

---
_Reported via `dacli report`._
- dacli: dev
- platform: darwin/arm64
- workspace and run transcript withheld (public upstream) — re-run with --disclose to include them

## Acceptance
- [x] `dacli github sync <project> --dry-run` validates the complete shared flag set before either the pull or push half can mutate local or remote state.
- [x] A valid dry-run previews both inbound adoptions and outbound task/decision/finding effects while leaving tasks, events, mappings, notes, GitHub issues, and comments unchanged.
- [x] An unknown or conflicting flag fails with exit 2 before any GitHub request that can write and before any local workspace write.
- [x] The real non-dry-run sync still performs pull then push, including mirroring a freshly adopted issue in the same invocation.
- [x] Shared sync flags such as task refs, `--since`, `--findings-as-issues`, and `--with-tasks` retain their documented forwarding behavior.
- [x] Tests snapshot task/event/mapping state and fake remote writes around public `github sync --dry-run`; mutation proof shows deferring validation until after pull makes the regression fail.
- [x] CLI/MCP help and GitHub/operator documentation derive or validate the exact preview signature.
- [x] `gofmt -l .`, `go vet ./...`, pinned `golangci-lint run`, and `go test ./...` pass.
## Log
- 2026-08-22T16:59:56Z accepted by a-root
- 2026-08-22T16:59:56Z verified by `GOCACHE=/private/tmp/dacli-go-cache go test ./docs ./internal/features/ghmirror -run 'TestPublicSupportClaimsMatchShippedSurface|TestSyncDryRunPreviewsBothHalvesAndWritesNothing|TestSyncStillRefusesATypo|TestSyncAcceptsPushOnlyFlags|TestPullAloneStillRefusesPushOnlyFlags' -count=1` (exit 0) in branch main at 300d1ae — proves that tree builds, not that the work is in trunk
- 2026-08-22T16:59:56Z deliverable: dacli/491-agent-report-github-sync-dry-run-mutates-by-adopting-issues-before-rejecting is merged into main
- 2026-08-22T16:59:56Z completed by a-root
- 2026-08-22T17:21:42Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/773 (event 01M0N65QNES47N3SNF3P6GSS8J)
- 2026-08-22T17:21:42Z a-root: Landing policy override: mode=pr base=main (event 01M0N6KK9W0PY5WJWQ79E7SSM4)
- 2026-08-22T17:21:42Z a-root: Integrated via PR https://github.com/mlnomadpy/dacli/pull/773 at merge commit 300d1ae518c349783a1b3d095c1abee89d064cc5 into main (event 01M0N6KTQRH76EBRXDHKZM1RFY)
## Verification Evidence
{"command":"GOCACHE=/private/tmp/dacli-go-cache go test ./docs ./internal/features/ghmirror -run 'TestPublicSupportClaimsMatchShippedSurface|TestSyncDryRunPreviewsBothHalvesAndWritesNothing|TestSyncStillRefusesATypo|TestSyncAcceptsPushOnlyFlags|TestPullAloneStillRefusesPushOnlyFlags' -count=1","exit_code":0,"duration_ms":826,"artifact_hash":"sha256:ca6720058fc99d95489c5efa743c0ec175332a93e52d3885fc1c775453aa0e2e","verifier":"a-root","branch":"main","commit_sha":"300d1ae518c349783a1b3d095c1abee89d064cc5"}
{"command":"GOCACHE=/private/tmp/dacli-go-cache go test ./docs ./internal/features/ghmirror -run 'TestPublicSupportClaimsMatchShippedSurface|TestSyncDryRunPreviewsBothHalvesAndWritesNothing|TestSyncStillRefusesATypo|TestSyncAcceptsPushOnlyFlags|TestPullAloneStillRefusesPushOnlyFlags' -count=1","exit_code":0,"duration_ms":1067,"artifact_hash":"sha256:7b5a50c2bff6088e3c2bec45bf40008116b6adfe545984f7381d9553f606784f","verifier":"a-root","branch":"main","commit_sha":"300d1ae518c349783a1b3d095c1abee89d064cc5"}
