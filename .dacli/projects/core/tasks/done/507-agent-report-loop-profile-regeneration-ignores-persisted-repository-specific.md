---
id: t-01M1068M1FTM2MHQ9WJ12CK9MV
kind: task
created: 2026-08-26T23:25:11Z
created_by: a-root
owner: a-root
github:
  issue: 801
  repo: mlnomadpy/dacli
estimate: "{optimistic: 5, probable: 8, pessimistic: 13}"
---
# [agent-report] loop profile regeneration ignores persisted repository-specific policy
## Context
Adopted from GitHub issue #801.

On a public Python/Vue/TypeScript monorepo, dacli start --project periodica --profile loop --width 1 --configure persisted .dacli/profiles/periodica.json with Go-only verification commands (gofmt, go vet, golangci-lint, go test) and auto_merge=true, despite the linked project map listing Python and TypeScript and the repository landing policy targeting protected dev. After correcting the persisted JSON to the actual backend/SDK/frontend commands and auto_merge=false, dacli start --project periodica --show --json read the corrected policy, but dacli start --project periodica --profile loop --width 1 --dry-run regenerated the same Go/auto-merge defaults instead of resolving the persisted policy. A bare dacli loop also resumed the stale legacy journal and attempted work before displaying usage. Expected: configure detects the repository stack or refuses ambiguous verification; subsequent explicit profile resolution merges persisted project policy with CLI overrides; auto-merge defaults false unless repository capability/policy enables it; dry-run and execution resolve the same policy. Acceptance: regression fixture for a Python+Vue monorepo; persisted verification and landing settings survive --profile loop --width overrides; unknown stack fails closed; no execution path silently replaces persisted commands with Go defaults.

---
_Reported via `dacli report`._
- dacli: dev
- platform: darwin/arm64
- workspace and run transcript withheld (public upstream) — re-run with --disclose to include them

## Acceptance
- [x] A Python/Vue monorepo fixture configures repository-specific verification commands and `auto_merge=false`, then resolves the same values after `--profile loop --width 1` overrides.
- [x] Explicit CLI profile fields override only their own persisted fields; omitted fields are never regenerated from Go defaults.
- [x] Stack detection selects commands supported by the adopted codebase map, and an unknown or ambiguous stack refuses with a configuration remedy.
- [x] Dry-run and real profile execution consume one resolved policy and report identical verification and landing settings.
- [x] A bare loop cannot silently resume a stale journal that contradicts the current persisted project profile.
- [x] Mutation evidence and the full repository verification gates pass, including non-Go regression fixtures.
## Log
- 2026-08-27T10:55:53Z accepted by a-root
- 2026-08-27T10:55:53Z verified by `GOCACHE=/private/tmp/dacli-go-cache-507 go test ./...` (exit 0) in branch main at 69d73e1 — proves that tree builds, not that the work is in trunk
- 2026-08-27T10:55:53Z deliverable: dacli/507-agent-report-loop-profile-regeneration-ignores-persisted-repository-specific is merged into main
- 2026-08-27T10:55:53Z completed by a-root
- 2026-08-27T11:01:04Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/809 (event 01M11D0E9Q96HQ2CAC5PF4JZCF)
## Verification Evidence
{"command":"GOCACHE=/private/tmp/dacli-go-cache-507 go test ./...","exit_code":0,"duration_ms":1595,"artifact_hash":"sha256:07db5a7f494d86162ce19dd327d134bb8ac313296b8ab5376f50a81125baa384","verifier":"a-root","branch":"dacli/507-agent-report-loop-profile-regeneration-ignores-persisted-repository-specific","commit_sha":"a1595d183277ffb417407f2db6c203aa8866fcf6"}
{"command":"GOCACHE=/private/tmp/dacli-go-cache-507 go test ./...","exit_code":0,"duration_ms":1729,"artifact_hash":"sha256:07db5a7f494d86162ce19dd327d134bb8ac313296b8ab5376f50a81125baa384","verifier":"a-root","branch":"main","commit_sha":"69d73e1f273110da9ddf501a7fbc92e038582d9d"}
