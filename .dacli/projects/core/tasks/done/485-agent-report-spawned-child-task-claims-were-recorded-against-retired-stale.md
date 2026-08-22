---
id: t-01M0D2KPMXPN55ASMH9XQXY21J
kind: task
created: 2026-08-19T13:15:45Z
created_by: a-root
owner: a-root
github:
  issue: 725
  repo: mlnomadpy/dacli
estimate: "{optimistic: 5, probable: 8, pessimistic: 13}"
depends_on: "[t-01M0AK4XK4M7CTJ6DXRKFW8XWG]"
---
# Bind spawned task claims to the newly minted child identity
## Context
Adopted from GitHub issue #725.

## Reproduction

Three detached spawns minted new children but wrote each task's claim event with
a retired child from an earlier run of the same role:

- task 012 spawned `a-codex-fixer-terra-c1rexx` in run `01M0CZFYNP`, but the
  task log named retired `a-codex-fixer-terra-hqf9ek`;
- task 013 spawned `a-codex-fixer-terra-bmjzgy` in run `01M0CZHA7C`, but the
  task log named retired `a-codex-fixer-terra-3gfdpn`;
- task 014 spawned `a-codex-fixer-1a6ne8` in run `01M0CZG5GD`, but the task
  log named retired `a-codex-fixer-ssj5r3`.

The stale identity then consumed WIP capacity and caused the actual child's
claim-scoped `dacli commit` to refuse valid files. The manual recovery retired
the stale identities and reconciled the real run IDs at root.

## Design direction

Treat child creation as one identity transaction. The child ID minted for a
spawn must be passed explicitly through claim creation, run recording,
`proc.txt`, and commit authorization; no step may rediscover a child by role or
"latest" roster entry. Persist enough linkage to diagnose a partial spawn
without assigning work to a previous identity.

## Acceptance
- [x] A fixture with a retired child and a new child of the same role records the task claim against the newly minted child ID.
- [x] When `dacli spawn` returns success, the task log, run record, `proc.txt`, and `agent show` contain the minted child ID and returned run ID.
- [x] WIP accounting counts the new live child once and does not count the retired identity as active work.
- [x] Claim-scoped `dacli commit` authorizes the new child for its declared paths and does not require retiring an unrelated historical child.
- [x] Two concurrent same-role spawns receive distinct identities: task A retains child A's claim and task B retains child B's claim.
- [x] A failed or partially recorded spawn cannot leave a task claimed by a pre-existing agent selected through role/name lookup.
- [x] Mutation evidence makes the regression test fail when claim attribution is changed back to role-based or stale-agent lookup.
- [x] `gofmt -l .`, `go vet ./...`, the pinned `golangci-lint run`, and `go test ./...` pass.
## Log
- 2026-08-22T14:43:03Z dependency edit by a-root (event 01M0MYSPACQ047RTM47306XBGB)
- 2026-08-22T16:58:57Z accepted by a-root
- 2026-08-22T16:58:57Z verified by `GOCACHE=/private/tmp/dacli-go-cache go test ./internal/features/execution ./internal/store -run 'TestClaimTask|Test.*Claimed' -count=1` (exit 0) in branch main at 58542b4 — proves that tree builds, not that the work is in trunk
- 2026-08-22T16:58:57Z deliverable: dacli/485-agent-report-spawned-child-task-claims-were-recorded-against-retired-stale is merged into main
- 2026-08-22T16:58:57Z completed by a-root
- 2026-08-22T17:21:42Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/772 (event 01M0N65K549DDRQWXKCPKZX0Q2)
- 2026-08-22T17:21:42Z a-root: Landing policy override: mode=pr base=main (event 01M0N6HK9BCW34JGP135WKGYSP)
- 2026-08-22T17:21:42Z a-root: Integrated via PR https://github.com/mlnomadpy/dacli/pull/772 at merge commit 58542b4e375659aaec18192d711005995aed37d9 into main (event 01M0N6HVHJSRAG6AMM91K3FT5Y)
## Verification Evidence
{"command":"GOCACHE=/private/tmp/dacli-go-cache go test ./internal/features/execution ./internal/store -run 'TestClaimTask|Test.*Claimed' -count=1","exit_code":0,"duration_ms":621,"artifact_hash":"sha256:66cd9b55a6eefa6762e2fece616ffae8c5b1922ebdef49bfeecb21755bc352ca","verifier":"a-root","branch":"main","commit_sha":"58542b4e375659aaec18192d711005995aed37d9"}
{"command":"GOCACHE=/private/tmp/dacli-go-cache go test ./internal/features/execution ./internal/store -run 'TestClaimTask|Test.*Claimed' -count=1","exit_code":0,"duration_ms":604,"artifact_hash":"sha256:88148bde52fee1d87b2e7ab75e6c980a59c5a770e34c2846884d2d975b3c4f5c","verifier":"a-root","branch":"main","commit_sha":"58542b4e375659aaec18192d711005995aed37d9"}
