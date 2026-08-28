---
id: t-01M147R9WKDPHMMT07HHC6V37B
kind: task
created: 2026-08-28T13:08:11Z
created_by: a-root
owner: a-root
github:
  issue: 871
  repo: mlnomadpy/dacli
estimate: "{optimistic: 1, probable: 2, pessimistic: 3}"
depends_on: "[t-01M11HZW8X678270CFWDBK7YTN, t-01M12QX9HEPKAAS1033W6HS45D, t-01M146B9RBYKTG8XCHNMZ72T9K, t-01M146B9TFB8Y6CX6FMMK91J9Q, t-01M146B9WDE7025DPT49EQKV0J, t-01M146B9YAADZ9X23NQ9EFE55N, t-01M146BA07Z5BTS3TTB2ADW7D4, t-01M146BA26TEH86YE16XHDZKGY, t-01M146BA434CD4A9778E8BNJ61, t-01M1493JBGSSHWDXAVAY5W7B9E, t-01M1493JDJSHRPAE5QWE87CEWD, t-01M1493JFF0JAMR7ZW3G90DKBJ, t-01M1493JHEAX82WX246TPCDR43, t-01M1493JKBNQCJRES2HE15X3BF, t-01M1493JN9F1F22S30ZW9HCVNC, t-01M1493JQ62NY4E39PQHNEZ6TP]"
---
# [agent-report] Product feedback: strengthen reconciliation, partial-task PRs, and operator ergonomics
## Context
Adopted from GitHub issue #871.

## Overall assessment

dacli has a strong foundation as an agent control plane. Its most valuable qualities are durable state, bounded autonomy, isolated worktrees, explicit ownership, recovery, critical-path planning, and an integrated GitHub landing workflow.

The largest reliability gap is reconciliation: local task state, branch heads, pull requests, CI results, and merged commits can disagree, and stale task-to-PR metadata can be trusted over current GitHub state.

## Features that worked especially well

- Bounded loops with explicit cycle and token limits.
- Durable tasks, findings, decisions, ownership, and execution history.
- Isolated worktrees and path claims.
- Audited root reclaim of terminal worker worktrees.
- Task-linked, attributed commits through dacli commit.
- Integrated push, PR, status, integration, and cleanup capabilities.
- Dry runs and explicit policy refusals.
- Critical-path ordering, estimates, and dependency support.
- Separation between harness selection, model routing, and project policy.
- Preservation of unique unmerged work while pruning completed work.

## Highest-priority reliability issues

### 1. Exact task-to-PR reconciliation

When a long-running task produces multiple PRs from a reused canonical branch, status and integration can select an older merged PR instead of the newest open PR. This can produce a false already-landed result and can trigger cleanup based on stale evidence.

PR identity should require repository, exact head branch, exact head SHA, base branch, and lifecycle state. Open PRs should take precedence over historical merged PRs. A worktree should only be cleaned when its current head is proven reachable from the target branch.

### 2. First-class partial-task slices

Large tasks frequently require several independently reviewable PRs. Please add a Task -> Slice -> PR model where every slice has its own branch, acceptance evidence, PR, merge SHA, and cleanup lifecycle. The parent task and issue should close only when all required slices and parent acceptance criteria are complete.

Suggested interface:

- dacli slice add --task REF --title TITLE
- dacli pr --slice REF/SLICE
- dacli task progress REF

### 3. Public-safe GitHub projection

Partial PRs should use Refs rather than Fixes by default. Internal findings, agent identities, verdicts, journals, and recovery details should remain private unless explicitly approved.

Suggested modes:

- dacli pr --partial
- dacli pr --closes-issue
- dacli github push --visibility public-safe
- dacli github push --include-findings

Publishing internal findings or decision issues to a public repository should be opt-in.

### 4. Worker capability preflight and structured handoff

Workers can successfully edit and verify source code while lacking permission to write Git or dacli metadata. Such runs should not be summarized as produced nothing.

Preflight should test source writes, worktree Git metadata, dacli event writes, network availability when required, and runtime/package-manager availability.

When commit access is unavailable, emit a structured root handoff containing changed paths, verification evidence, unresolved findings, and the required next action.

### 5. Acceptance migration for adopted tasks

Adopted issues may contain prose acceptance criteria but no structured acceptance checkboxes, causing acceptance to fail after implementation is complete.

Please automatically convert suitable acceptance bullets during adoption, or provide:

- dacli task acceptance migrate REF --from-section "Acceptance criteria"

## Additional improvements

- Add dacli reconcile to compare tasks, issues, branches, worktrees, PRs, exact head SHAs, CI, and trunk reachability.
- Add dacli pr wait --checks and dacli pr land --when-green.
- Add dacli branches audit and dacli branches prune --merged --dry-run.
- Preserve underlying Git stderr in failures instead of only reporting exit status.
- Support --help consistently on compound commands.
- Make --json output consistent across commands.
- Reconcile loop rollups when work lands outside the original loop invocation.
- Automatically repair or compact malformed and obsolete event-journal entries.
- Show spent, remaining, and reserved integration/recovery tokens in loop status.
- Ingest CI checks and artifacts as structured evidence attached to the exact commit.
- Provide a supported reconciliation path after a manual GitHub fallback.

## CLI surface simplification

The primary workflow should be extremely clear:

inspect -> plan -> work -> verify -> PR -> checks -> merge -> reconcile -> clean

Features such as glossary management, catalogs, generic queues, dashboards, calibration, skill promotion, templates, decision-issue publication, and detailed verdict publication can remain available but should be grouped as advanced, administrative, or plugin capabilities.

There is also conceptual overlap among start, loop, ship, wave, supervise, and task mode. Making dacli start --profile the primary interface and retaining the others as expert shortcuts would improve discoverability.

## Recommended roadmap order

1. Fix exact PR/head-SHA reconciliation and cleanup invariants.
2. Add multi-slice tasks with multiple PRs.
3. Add public-safe GitHub projection and partial-PR semantics.
4. Add worker capability preflight and structured root handoff.
5. Add one authoritative reconcile command.
6. Automatically structure adopted acceptance criteria.
7. Simplify the default CLI surface.
8. Add native CI waiting, landing, and branch auditing.

With these changes, dacli would be much easier to trust for sustained autonomous project loops without requiring manual GitHub verification after each landing decision.

---
_Reported via `dacli report`._
- dacli: dev
- platform: darwin/arm64
- workspace and run transcript withheld (public upstream) — re-run with --disclose to include them

## Acceptance
## Log
- 2026-08-28T13:32:47Z dependency edit by a-root (event 01M1495BZ0WRGWJK7DHXZRZQ4F)
