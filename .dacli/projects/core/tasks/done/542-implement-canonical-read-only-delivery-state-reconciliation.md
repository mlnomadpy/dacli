---
id: t-01M146BA62817V08T9P6D6REKT
kind: task
created: 2026-08-28T12:43:37Z
created_by: a-root
owner: a-root
github:
  issue: 856
  repo: mlnomadpy/dacli
estimate: "{optimistic: 8, probable: 13, pessimistic: 21}"
---
# Implement canonical read-only delivery-state reconciliation
## Context
Adopted from GitHub issue #856.

## Parent

Part of #855.

## Observed symptom

DACLI exposes task, agent, run, worktree, event, loop, branch, PR, CI, and trunk state through separate commands. Those surfaces can disagree without one command identifying the inconsistencies. On the dacli repository itself, `status` reported pending events, `agents` reported no live agents, `loop status` retained a stale landing/trunk checkpoint, GitHub had no open PR, and `doctor` still reported no anti-patterns.

## Objective

Implement a canonical, read-only delivery-state projection and expose it through:

```bash
dacli reconcile --project <project> --dry-run
dacli reconcile --project <project> --json
```

The projection is the shared source for future repair, loop recovery, PR diagnosis, and cleanup. It must derive facts from existing records rather than introduce another mutable lifecycle ledger.

## Required classifications

- Task intent/status and owner versus live agent/run state.
- Task branch/worktree existence, cleanliness, and claimed paths.
- Canonical PR identity, open/closed/merged state, head/base SHA, and required checks.
- Pending loop landings and trunk marker versus freshly observed configured base.
- Pending/refused/unresolved events, including events targeting terminal or missing records.
- Blocked/ready scheduling inconsistencies and unresolved dependencies.
- Unknown/unobservable external state as unknown, never success.

## Architectural constraints

- Put the reusable projection/classification below feature slices so `doctor`, loop recovery, PR diagnosis, and cleanup can consume it without slice imports.
- Keep the first implementation read-only. No repair or deletion belongs in this task.
- Define stable typed classifications and a versioned JSON schema; human output is a rendering of the same model.
- GitHub authentication, network failure, and incomplete evidence must fail closed and remain distinguishable.
- Project scoping must not imply a separate Git history or security boundary.



## Non-goals

- Applying repairs.
- Deleting branches or worktrees.
- Persisting every derived delivery stage as task status.
- Creating a hosted control plane.

## Manual workaround today

Operators run `status`, `doctor`, `agents`, `runs list`, `worktree list`, `loop status`, `events tail`, `pr status`, and fresh git/GitHub inspection, then reconcile the results manually.

## Acceptance
- [x] A fixture containing an orphaned active task, finished unfinalized run, stale loop landing, terminal-task event, and closed-unmerged PR produces the expected distinct classifications in text and JSON.
- [x] `dacli reconcile --project core --dry-run` performs no filesystem, event-log, git, or GitHub mutation; a before/after digest test proves this.
- [x] The JSON output has a schema/version field, stable object identifiers, observed source, observed time, severity, confidence/unknown state, and suggested next action.
- [x] GitHub outage/authentication fixtures report external state as unknown and exit nonzero or policy-refused according to the CLI contract; they never report reconciled success.
- [x] `doctor` can call the shared local-only classifier without importing a feature slice, with an architecture test protecting the boundary.
- [x] Regression tests fail when any one fixture inconsistency is suppressed from the projection.
## Log
- 2026-08-28T12:47:45Z claimed by a-maintainer-k0gy77
- 2026-08-28T13:38:41Z accepted by a-root
- 2026-08-28T13:38:41Z verified by `GOCACHE=/tmp/dacli-542-accept-cache go test ./internal/store ./internal/features/reconciliation -count=1` (exit 0) in branch dacli/542-implement-canonical-read-only-delivery-state-reconciliation at d0442b5f — proves that tree builds, not that the work is in trunk
- 2026-08-28T13:38:41Z completed by a-root
- 2026-08-28T13:52:57Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/879 (event 01M1491X8YXV5ESYM1ZRRGHPQH)
- 2026-08-28T13:52:57Z a-root: Landing policy override: mode=pr base=main (event 01M149GKRTXWT0Y4M1CXBTRF8J)
- 2026-08-28T13:52:57Z a-root: Integrated via PR https://github.com/mlnomadpy/dacli/pull/879 at merge commit 428931a5ef13dcf93f6affdf639b0ea7307f18b6 into main (generation 0) (event 01M149GTVE0WSW1QJG5W4ANRBP)
## Verification Evidence
{"command":"GOCACHE=/tmp/dacli-542-owner-cache go test ./... -count=1","exit_code":0,"duration_ms":59917,"artifact_hash":"sha256:a01ecb6f129e7e7daa22d3695b061cae11f3bcb2fd9613cc080c717d2e44f669","verifier":"a-root","branch":"dacli/542-implement-canonical-read-only-delivery-state-reconciliation","commit_sha":"bc91d118fd3fe7ad718416608550ec00d3910b8d"}
{"command":"go test ./internal/store ./internal/features/reconciliation -count=1","exit_code":0,"duration_ms":30749,"artifact_hash":"sha256:734bcb24b4e8dbd2bd614e3c82fb503cfedd06940000d8a28ee7c5d45e93a3e8","verifier":"a-root","branch":"dacli/542-implement-canonical-read-only-delivery-state-reconciliation","commit_sha":"bc91d118fd3fe7ad718416608550ec00d3910b8d"}
{"command":"GOCACHE=/tmp/dacli-542-owner-cache go test ./... -count=1","exit_code":0,"duration_ms":61390,"artifact_hash":"sha256:384985ec5f8afcfb4b026ff067b344ce2fd53e8d2fcc73a734fae526b0d2c5fd","verifier":"a-root","branch":"dacli/542-implement-canonical-read-only-delivery-state-reconciliation","commit_sha":"d0442b5fbaeda5483dc29c565db727a7fead88f1"}
{"command":"GOCACHE=/tmp/dacli-542-accept-cache go test ./internal/store ./internal/features/reconciliation -count=1","exit_code":0,"duration_ms":32770,"artifact_hash":"sha256:84cec064ad54d061af1d209aea18c54f1f68b5c014c2ebf3c21501af5306338e","verifier":"a-root","branch":"dacli/542-implement-canonical-read-only-delivery-state-reconciliation","commit_sha":"d0442b5fbaeda5483dc29c565db727a7fead88f1"}
