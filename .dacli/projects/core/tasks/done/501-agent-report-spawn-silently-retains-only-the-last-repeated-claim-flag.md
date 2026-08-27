---
id: t-01M0ZCAPQAVDTVV82DNRW7969Q
kind: task
created: 2026-08-26T15:51:56Z
created_by: a-root
owner: a-root
github:
  issue: 794
  repo: mlnomadpy/dacli
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
---
# [agent-report] spawn silently retains only the last repeated claim flag
## Context
Adopted from GitHub issue #794.

Reproduced twice with detached isolated-worktree spawns. Commands supplied two explicit --claim flags, but the resulting run retained only the second path. Implementations correctly modified files under the first claim, then dacli commit refused with policy exit 3; owner recovery required worktree reclaim with exact paths. Expected: repeated --claim flags accumulate, or the CLI rejects repeated syntax and documents a single comma-separated form before spawning. Acceptance criteria: a spawn with --claim path-a --claim path-b records both claims; commit accepts files under either and rejects a third path; the resolved claims are printed in dry-run/advise output. Non-goal: weakening claim enforcement.

---
_Reported via `dacli report`._
- dacli: dev
- platform: darwin/arm64
- workspace and run transcript withheld (public upstream) — re-run with --disclose to include them

## Acceptance
- [x] `dacli spawn --claim path-a --claim path-b` records both normalized claims instead of retaining only the last flag.
- [x] `dacli commit` accepts files under either recorded claim and refuses a third unclaimed path with policy exit 3.
- [x] Spawn dry-run or advise output prints the complete resolved claim set.
- [x] Repeated and comma-separated claim forms have documented, regression-tested semantics without weakening overlap enforcement.
- [x] Mutation evidence and the full repository verification gates pass.
## Log
- 2026-08-27T09:55:44Z accepted by a-root
- 2026-08-27T09:55:44Z verified by `GOCACHE=/private/tmp/dacli-go-cache-501-main go test ./...` (exit 0) in branch main at 789712b — proves that tree builds, not that the work is in trunk
- 2026-08-27T09:55:44Z deliverable: dacli/501-agent-report-spawn-silently-retains-only-the-last-repeated-claim-flag is merged into main
- 2026-08-27T09:55:44Z completed by a-root
- 2026-08-27T09:57:07Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/806 (event 01M119X04RA2NY3FNDE6M3S2QZ)
## Verification Evidence
{"command":"GOCACHE=/private/tmp/dacli-go-cache-501 go test ./internal/features/execution -run TestResolveLaunchAccumulatesRepeatedAndCommaSeparatedClaims -count=1","exit_code":0,"duration_ms":603,"artifact_hash":"sha256:f762722a5c577d8969a49c8bc794548f7866dd981b3908c9c0cacbfbb6f8273a","verifier":"a-root","branch":"dacli/501-agent-report-spawn-silently-retains-only-the-last-repeated-claim-flag","commit_sha":"a4b873397406bb0f9c25c5603eafa59dd0c54391"}
{"command":"GOCACHE=/private/tmp/dacli-go-cache-501 go test ./internal/features/vcs -run TestCommitAcceptsTrailingRecursiveClaimDescendantsOnly -count=1","exit_code":0,"duration_ms":621,"artifact_hash":"sha256:b6d660e0b3420d21dd302b6ab3b6af12afe39bcb16e2e6f7267471ea3ea97687","verifier":"a-root","branch":"dacli/501-agent-report-spawn-silently-retains-only-the-last-repeated-claim-flag","commit_sha":"a4b873397406bb0f9c25c5603eafa59dd0c54391"}
{"command":"GOCACHE=/private/tmp/dacli-go-cache-501 go test ./internal/features/execution -run TestSpawnAdviseAccumulatesRepeatedAndCommaSeparatedClaims -count=1","exit_code":0,"duration_ms":433,"artifact_hash":"sha256:f762722a5c577d8969a49c8bc794548f7866dd981b3908c9c0cacbfbb6f8273a","verifier":"a-root","branch":"dacli/501-agent-report-spawn-silently-retains-only-the-last-repeated-claim-flag","commit_sha":"a4b873397406bb0f9c25c5603eafa59dd0c54391"}
{"command":"GOCACHE=/private/tmp/dacli-go-cache-501 go test ./internal/features/execution -run 'Test(SpawnAdvise|ResolveLaunch)AccumulatesRepeatedAndCommaSeparatedClaims' -count=1","exit_code":0,"duration_ms":464,"artifact_hash":"sha256:7b91e9fdb8d3cb49e1655028d1c365a65aeeaf9197f5f0a60dfef0bf0a8d159c","verifier":"a-root","branch":"dacli/501-agent-report-spawn-silently-retains-only-the-last-repeated-claim-flag","commit_sha":"a4b873397406bb0f9c25c5603eafa59dd0c54391"}
{"command":"GOCACHE=/private/tmp/dacli-go-cache-501 go test ./...","exit_code":0,"duration_ms":1800,"artifact_hash":"sha256:07db5a7f494d86162ce19dd327d134bb8ac313296b8ab5376f50a81125baa384","verifier":"a-root","branch":"dacli/501-agent-report-spawn-silently-retains-only-the-last-repeated-claim-flag","commit_sha":"a4b873397406bb0f9c25c5603eafa59dd0c54391"}
{"command":"GOCACHE=/private/tmp/dacli-go-cache-501-main go test ./...","exit_code":0,"duration_ms":68265,"artifact_hash":"sha256:0c430b14e8a356266dba7bf9a6531625ca9accf7f9bf6daab12dc7f7acd1f1b9","verifier":"a-root","branch":"main","commit_sha":"789712b866e489b0a5e3cc371e14cfb825822824"}
