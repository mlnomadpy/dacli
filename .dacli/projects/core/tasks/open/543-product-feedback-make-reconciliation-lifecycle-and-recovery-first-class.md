---
id: t-01M146BA81VWWJ27PWFRFN1E2M
kind: task
created: 2026-08-28T12:43:37Z
created_by: a-root
owner: a-root
github:
  issue: 855
  repo: mlnomadpy/dacli
estimate: "{optimistic: 1, probable: 2, pessimistic: 3}"
depends_on: "[t-01M146B9RBYKTG8XCHNMZ72T9K, t-01M146B9TFB8Y6CX6FMMK91J9Q, t-01M146B9WDE7025DPT49EQKV0J, t-01M146B9YAADZ9X23NQ9EFE55N, t-01M146BA07Z5BTS3TTB2ADW7D4, t-01M146BA26TEH86YE16XHDZKGY, t-01M146BA434CD4A9778E8BNJ61, t-01M146BA62817V08T9P6D6REKT]"
---
# Product feedback: make reconciliation, lifecycle, and recovery first-class
## Context
Adopted from GitHub issue #855.

## Context

This feedback comes from sustained use of DACLI for a multi-language monorepo with task worktrees, independent reviews, PR landing, GitHub Actions, and bounded autonomous loops. The core model is strong: durable evidence, explicit authority, isolated work, and fail-closed landing are materially better than an ungoverned agent swarm.

The main problem is operational state drift. When agents, tasks, worktrees, PRs, CI, pending events, and the loop journal disagree, the operator currently has to reconstruct the truth through many commands. At that point, managing DACLI can consume more effort than managing the repository.

This is a consolidated field-feedback issue rather than a claim that every item
is new. Related completed work includes #188 (PR/auto-merge progress), #606
(structured verification provenance), #699 (stale loop proposals), and #801
(persisted repository-specific loop policy). The request here is to make the
end-to-end recovery experience coherent across those individual mechanisms.

## Features that worked especially well

- Durable tasks with checkable acceptance criteria, blockers, decisions, and verification evidence.
- Isolated worktrees and canonical task branches.
- Governed commits with `Dacli-Agent`, `Dacli-Role`, and `Dacli-Task` trailers.
- Fail-closed PR landing: failed CI, missing evidence, authentication outages, and unknown landing state are not treated as success.
- Bounded loops, STOP support, token budgets, and a no-progress breaker.
- `task block`, `task takeover`, `github doctor`, and `pr status` as explicit recovery primitives.
- Independent implementation/review roles; adversarial review found defects that ordinary tests missed.
- Critical-path and estimate support when the backlog and dependency graph are coherent.

The most valuable DACLI principle is: **failure to verify is not success**. Please preserve it.

## Highest-priority gaps

### 1. Add a first-class reconciliation command

Proposed interface:

```bash
dacli reconcile --dry-run
dacli reconcile --apply-safe
```

It should compare and classify:

- task status and owner;
- live agents and completed runs;
- worktrees and branches;
- open, closed, and merged PRs;
- pending loop landings and trunk markers;
- pending/refused/unresolved events;
- required CI state.

Example output:

```text
14 active tasks have no live agent
2 loop landings reference a closed or missing PR
1 blocked task is still counted as ready
30 events target records that no longer resolve
```

The command should propose safe repairs and require explicit authority for destructive or ambiguous changes.

### 2. Define one explicit task lifecycle

The distinction between `task done`, `accept`, `ship`, `integrate`, and merged delivery is difficult to reason about. A task can currently be locally "done" while its PR is open and CI has never run.

Suggested state model:

```text
planned -> claimed -> implemented -> locally_verified -> reviewed
        -> pr_open -> ci_green -> merged -> done
```

`blocked` should be usable from any nonterminal stage. For PR-mode projects, `done` should normally mean the exact reviewed commit landed on the configured base branch.

### 3. Bind verification to the final committed tree

Verification recorded before a commit can refer to the parent commit even when the working tree was what actually passed. DACLI should either verify after committing or assert that the final commit tree matches a verified tree hash.

Evidence should bind:

- commit SHA and tree SHA;
- commands and exit codes;
- environment/tool versions;
- reviewer verdicts;
- GitHub run/check IDs;
- explicitly skipped external evidence.

### 4. Treat no response as unknown, never refuted

An independent reviewer returning no output or no verdict must not count as evidence that a finding is refuted. Suggested states:

- confirmed;
- refuted with evidence;
- inconclusive;
- no response;
- infrastructure failure.

Only an evidence-bearing refutation should reduce confidence in an implementation finding.

### 5. Make loop recovery self-explanatory and resumable

The loop should emit a structured halt diagnosis rather than retaining stale pending landings and a ready count inconsistent with `next`.

Example:

```text
HALTED_EXTERNAL
- required GitHub checks could not start because of an account-level restriction
- one recorded PR was closed without merging
- one pending task has no canonical PR
- no open unclaimed task is schedulable
```

It should resume after observing the external condition change or real trunk advancement. Operators should not need to reset the no-progress counter merely to recover.

### 6. Add native PR/CI diagnosis

Proposed command:

```bash
dacli pr diagnose --task <ref>
```

Classify at least:

- test or workflow failure;
- workflow syntax failure;
- runner unavailable;
- billing/spending restriction;
- authentication failure;
- GitHub outage;
- pending approval;
- branch conflict or stale base.

Check annotations often contain the actual cause and should be incorporated into task blocking and recovery guidance.

### 7. Negotiate skill/documentation compatibility with the installed CLI

The operating skill can recommend flags unsupported by the installed binary. Add something like:

```bash
dacli capabilities --json
dacli version --compatibility
```

Skills and generated guidance should branch on supported capabilities instead of assuming the latest interface.

### 8. Make verification monorepo-aware

Generic commands such as `python -m pytest`, `npm test`, and `npm run build` are insufficient when Python, Android, web, shared contracts, Terraform, and documentation have different working directories and gates.

The adopted codebase map should support path-to-verification routing, for example:

- backend changes -> environment sync, lint, tests, and migrations;
- web changes -> typecheck, tests, and production build;
- Android changes -> Gradle compile/unit/lint gates;
- shared contract changes -> every affected language binding and golden vector;
- docs changes -> offline link and workflow checks.

### 9. Bridge human approval into host safety systems

High-consequence changes may be explicitly authorized in a DACLI task while the execution host cannot recognize that authorization. DACLI should be able to emit a signed/scoped authorization artifact containing:

- task and approving identity;
- permitted paths and operations;
- risk disclosure;
- expiration;
- prohibited external side effects.

Execution hosts could consume this artifact without treating a broad DACLI task as unlimited authority.

### 10. Add safe repository cleanup

Proposed command:

```bash
dacli cleanup --dry-run
```

Classify branches/worktrees as merged, open-PR, dirty, unpushed, abandoned, blocked, or protected. Cleanup should be recoverable and must never delete unlanded work automatically.

## Complexity that could move out of the default interface

These features may still be valuable, but they make the main workflow harder to discover:

- dozens of narrow or task-specific roles;
- overlapping completion/landing commands;
- glossary, catalog, calibration, velocity, burndown, queues, and stages;
- mandatory estimation/model-economics machinery for small projects;
- broad bidirectional GitHub synchronization.

Consider a small default roster (planner, implementer, security implementer, reviewer, security reviewer, integration owner) and move advanced portfolio functions under an advanced namespace or optional plugin.

The default product journey should be obvious:

```text
inspect -> plan -> claim -> implement -> verify -> review -> PR -> CI -> merge
```

## Smaller usability improvements

- Print concise success output for state-mutating commands that are currently silent.
- Support parent help such as `dacli task --help` in addition to leaf-command help.
- Add `doctor --scope project` and separate immediate blockers from known metadata debt.
- Explain conflicts where loop routing rejects every role but `team assign` still selects one.
- Use the configured landing base consistently; do not silently fall back from `master` to `main`.
- Support structured verification command arrays instead of nested shell quoting.
- Add a portable `--workspace` option for integration and temporary worktrees.
- Provide one stable installed binary/path and report which binary generated skills and state.

## Desired product promise

DACLI should be the transaction coordinator for software delivery, rather than requiring operators to become experts in its internal state:

> Give DACLI a scoped task. It preserves authority, isolates implementation, collects evidence, obtains independent review, lands through required checks, recovers safely from interruptions, and identifies the exact human action needed when it cannot proceed.

The foundations are already strong. Reconciliation, lifecycle clarity, evidence binding, CI diagnosis, and version compatibility would make the biggest improvement to day-to-day trust and usability.

## Acceptance
## Log
- 2026-08-28T12:44:47Z dependency edit by a-root (event 01M146DFE6Q3RWAA5D5XTEFK6P)
