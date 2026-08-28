---
id: t-01M147RA8749AVXNXB142NS5KT
kind: task
created: 2026-08-28T13:08:11Z
created_by: a-root
owner: a-root
github:
  issue: 864
  repo: mlnomadpy/dacli
estimate: "{optimistic: 1, probable: 2, pessimistic: 3}"
depends_on: "[t-01M147R9YKAKFW35P82YSVHZ83, t-01M147RA0JHR6H2JN3H83RBAEH, t-01M147RA2EPR7BKWWZ3WS8CY44, t-01M147RA4B2C2NAH855VBNT2SJ, t-01M147RA6982P2FQXN64RZFPG4]"
---
# Product feedback from sustained dacli orchestration: make task shipping transactional and resumable
## Context
Adopted from GitHub issue #864.

## Context

This feedback comes from using dacli as the control plane for sustained development of Periodica, including bounded implementation loops, isolated worktrees, backend and frontend changes, GitHub PR landing, CI verification, release promotion from `dev` to `master`, task recovery, and acceptance reconciliation.

Observed with dacli dev, revision `73304b56b1bcfec0632a9dbc98f8a6c70031b103`.

At the latest checkpoint the workspace had 60 completed tasks, 24 open tasks, four blocked tasks, and one stale active task. Recent loops built and landed repository decomposition tasks 091–094 and frontend extraction task 040 through PRs #101–#111 in the product repository.

## What worked especially well

1. **Durable task and evidence ledger.** Acceptance criteria, ownership, dependencies, commits, verification commands, and completion history made interruption recovery dependable.
2. **Bounded autonomous loops.** Cycle limits, WIP width, timeouts, rolling token windows, stop files, and explicit landing targets made autonomy governable.
3. **Isolated worktrees and ownership.** Task worktrees prevented collisions. The audited `worktree reclaim` refusal and transfer flow protected worker-owned changes.
4. **Executable acceptance.** `accept --verify` tied completion to a command, branch, commit, exit code, duration, and artifact hash rather than trusting an agent summary.
5. **Critical-path scheduling.** Estimates, dependency edges, and `next` kept work directed toward meaningful blockers.
6. **Role/model/runtime separation.** The capability and WIP model is a strong foundation for economical routing.
7. **GitHub-centered landing.** PR-only delivery, explicit landing bases, required checks, merge confirmation, and branch cleanup fit normal engineering governance.
8. **Safety refusals.** Dacli correctly refused unsafe ownership violations and incomplete acceptance.

## Main friction

### 1. Landing is not yet one resumable transaction

The practical workflow repeatedly required separate manual reconciliation:

1. worker commits;
2. dacli pushes;
3. dacli opens or reuses a PR;
4. operator locates the PR;
5. operator watches GitHub checks;
6. operator runs `integrate --force`;
7. operator fetches the landing branch;
8. operator runs `accept --force --verify`;
9. operator syncs events and prunes branches.

The distinction between built, landed, and accepted is valuable, but advancing those states should be one idempotent, resumable operation.

### 2. GitHub lookup failures can be misclassified

During transient DNS/connection-reset failures, dacli sometimes reported that no PR existed even though the canonical branch was pushed and the PR existed. A fallback also appeared capable of comparing against the repository default rather than the configured landing base (`dev`).

External uncertainty should remain explicit: PR absent, GitHub unreachable, authentication failed, or lookup inconclusive are different states. A failed lookup must not become “PR absent.”

### 3. Parent/child structure does not imply aggregate scheduling

Task 035 already had child tasks 040–044, but the parent did not depend on them. Dacli therefore ranked the oversized 13.5-point parent for direct implementation instead of its ready children. The dependency graph had to be repaired manually.

A parent with open implementation children should be optionally modeled as an aggregate milestone, excluded from direct assignment, and completed from its accepted children.

### 4. Capacity enforcement can be bypassed inconsistently

`team assign 035` correctly refused because the task exceeded configured role capacity, while an explicitly configured loop could still preview or attempt the same task with `frontend-dev`. Capacity overrides should require an explicit reason, or dacli should propose decomposition.

### 5. Reviewer spawning repeatedly failed

Every cycle attempted the configured read-only reviewer, but the runtime refused because the requested token/isolation constraints could not be enforced. The impossible phase was retried every cycle and produced noise rather than review.

The whole cycle should be preflighted before implementation starts. Dacli also needs a reliably enforceable read-only reviewer runtime and a native correction turn from structured findings.

### 6. Worker observability is too coarse

Workers frequently appeared as “thinking / TRANSCRIPT-ACTIVE” with stale transcript tails and no useful process information while they were actually editing, testing, committing, or preparing a summary. Determining progress required inspecting the worktree, run log, and git history independently.

### 7. CLI conventions are inconsistent

Examples encountered:

- `status --project` was rejected;
- `next --critical-path` was rejected even though `next` reports critical-path ranking;
- `route --help` did not map intuitively to `team route`;
- recovery syntax such as `--claim path,path` required experimentation.

Project scoping, task references, help behavior, JSON output, and refusal remediation should be consistent across commands.

### 8. Pending and stale states need explanation

`sync` repeatedly reported “3 applied, 9 left pending” without explaining ownership or remediation. The workspace also reported an active task when no corresponding live agent existed.

Pending events and active work should expose actionable sub-states such as live, detached, stale, awaiting merge, awaiting owner, and awaiting human.

## Highest-priority improvements

### P0 — Transactional task shipping

Provide one resumable workflow covering:

claim → worktree → implement → verify → commit → push → PR → checks → review → correction → merge → landed verification → accept → cleanup

Each transition should be idempotent and externally reconcilable. Rerunning after interruption should continue from the last confirmed checkpoint.

### P0 — Truthful external-state reconciliation

Add a first-class `dacli reconcile` operation that repairs local state from git, GitHub branches, PRs, checks, merge commits, and live processes. Use bounded retry/backoff, persist PR identity immediately, honor the configured landing base, and never infer absence from an unavailable API.

### P0 — Reliable independent review

Support enforceable read-only isolation and structured reviewer results containing verdict, severity, file, line, evidence, and suggested verification. Findings should feed a bounded implementation correction loop.

### P1 — Aggregate tasks and automatic decomposition

- Add an aggregate/milestone task kind.
- Detect parents that are independently schedulable alongside their children.
- Offer an atomic dependency repair preview.
- For oversized tasks, propose a WBS with acceptance criteria, estimates, path claims, and dependency edges.

### P1 — Verification profiles

Repeated project commands should become named and versioned profiles, for example `backend` and `frontend`. Acceptance evidence should record both the profile version and resolved commands.

### P1 — Better status and explainability

A single status view should include current stage, current command, elapsed command time, changed files, last commit, token/cost use, PR URL, CI state, next transition, and required operator action.

An `explain <task>` command should state why a task is ranked or blocked, why a role was selected, which dependencies matter, and the exact safe next action.

### P1 — Native release trains

Support promotion workflows such as `dev` → `master`: create release PR notes from accepted tasks, wait for CI, merge, confirm equivalent trees, and clean merged branches.

### P2 — Webhook/service reconciliation and human inbox

- Let service mode react to GitHub webhook state rather than depending only on polling.
- Add a compact inbox for questions and decisions that genuinely require human authority, including impact and unblocked tasks.

## Features that could move out of the core path

These may remain valuable for larger organizations, but they should be optional or clearly “advanced” until the build-review-land path is effortless:

- glossary and catalog publishing;
- contribution and blame reporting;
- burndown and calibration views;
- skill/template authoring and promotion;
- wiki and GitHub Project synchronization;
- mandatory retro/doctor work after every small successful task.

Similarly, a review phase known by preflight to be impossible should not run by default.

## Suggested product focus

Preserve dacli’s strongest ideas: durable state, explicit authority, isolated worktrees, acceptance evidence, role routing, and GitHub-based landing.

The highest-leverage next goal is:

> Make a bounded task move from ready to independently reviewed, landed, accepted, and cleaned up through one reliable, resumable transaction.

If that path gains truthful GitHub reconciliation, enforceable review, aggregate-task handling, stable structured output, and strong observability, dacli becomes a genuine engineering control plane rather than a collection of coding-agent wrappers.

## Acceptance
## Log
- 2026-08-28T13:32:48Z dependency edit by a-root (event 01M1495CM2J05G0F1XTYJ3ZFMR)
