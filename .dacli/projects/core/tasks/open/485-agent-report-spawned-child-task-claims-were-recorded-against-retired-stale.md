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
- [ ] A fixture with a retired child and a new child of the same role records the task claim against the newly minted child ID.
- [ ] When `dacli spawn` returns success, the task log, run record, `proc.txt`, and `agent show` contain the minted child ID and returned run ID.
- [ ] WIP accounting counts the new live child once and does not count the retired identity as active work.
- [ ] Claim-scoped `dacli commit` authorizes the new child for its declared paths and does not require retiring an unrelated historical child.
- [ ] Two concurrent same-role spawns receive distinct identities: task A retains child A's claim and task B retains child B's claim.
- [ ] A failed or partially recorded spawn cannot leave a task claimed by a pre-existing agent selected through role/name lookup.
- [ ] Mutation evidence makes the regression test fail when claim attribution is changed back to role-based or stale-agent lookup.
- [ ] `gofmt -l .`, `go vet ./...`, the pinned `golangci-lint run`, and `go test ./...` pass.
## Log
- 2026-08-22T14:43:03Z dependency edit by a-root (event 01M0MYSPACQ047RTM47306XBGB)
