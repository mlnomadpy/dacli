# Changelog

All notable changes to dacli are recorded here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project aims
to follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **A deployable control-plane boundary in source.** The modular-monolith
  skeleton now includes strict configuration, checksummed PostgreSQL migrations,
  tenant-scoped domain and repository APIs, revocable device sessions, bounded
  project/environment routes, a durable signed-envelope inbox/outbox worker,
  and local Docker topology. Forced row-level security, composite tenant keys,
  append-only audits, stable pagination, idempotency, replay floors, leases,
  retries, and dead letters are executable invariants rather than architecture
  prose. Hosted deployment composition remains intentionally unimplemented.
- **A tested recovery and retention contract.** The v1 runbook covers authority,
  RPO/RTO assumptions, backup/restore ordering, rollback, signing-key rotation,
  tenant deletion, backup expiry, and incident response. Its disposable
  PostgreSQL 17.6 harness restores a consistent snapshot into an empty database
  and verifies the migration ledger, optimistic versions, audit/idempotency
  identities, cursor/replay state, dead letters, and two-tenant RLS isolation.
- **An investigation-first dashboard.** Durable activity, task and agent
  inspectors, exact delivery evidence, outcome analytics, attention queue,
  dependency graph, loop supervision, filtering, responsive layouts, deep-link
  routing, and fixture-backed desktop/mobile evidence now form one read-only
  operator surface.
- **Resumable agent handoffs and observability.** Read-only reviewers can create
  exact-tree acceptance proposals for owner application; task, agent, log, loop,
  and overview JSON surfaces expose bounded, freshness-sourced progress and
  typed degraded outcomes.
- **Safer GitHub-first delivery.** Task branches start from the configured
  landing base, PR publication records a recoverable transaction, adoption and
  reconciliation use typed plans, and ruleset checks participate in acceptance.

### Changed

- Switched the project license from MIT to
  [BSD 3-Clause](LICENSE), retaining the copyright notice and non-endorsement
  condition in source and binary redistributions.
- Moved expensive macOS process validation to the tag-only release gate while
  retaining Linux race, coverage, lint, security, cross-compile, clean-checkout,
  dashboard, and GoReleaser snapshot checks on pull requests.
- Replaced accidental wall-clock sleeps and durable fixture barriers with
  deterministic test clocks and scale seeding; the full suite remains broad
  while consuming fewer hosted-runner minutes.
- Updated the dacli skill, runtime guidance, operator playbook, trust model, and
  compatibility documentation around explicit harness selection, bounded loops,
  critical-path planning, model/cost routing, recovery, and GitHub landing.

### Security

- Writable agents now receive enforceable claimed-path sandboxes, while
  restricted workers use parent-mediated, identity-bound commits instead of
  receiving broader mutation authority.
- Review launch, verification evidence, and acceptance are bound to immutable
  trees and complete preflight contracts; stale or mismatched handoffs refuse.
- Tenant mutations bind a canonical action digest and explicit success/refusal/
  conflict/failure reason to verified actor, device, tenant, correlation, and
  versions. Authenticated failed attempts remain auditable after rollback.
- Independent bounded rate limits protect pre-auth identity verification,
  authenticated high-cardinality reads, sync ingestion, and delivery using only
  trusted direct-peer or verified-principal keys.

### Fixed

- Refused spawns now finalize before runtime start, redundant terminal PR events
  reconcile automatically, and a published canonical task branch cannot be
  mistaken for an unpublished or unrelated branch.
- Cleanup can safely classify detached acceptance worktrees, reviewer results
  survive restricted handoff, and sequence-lock tests no longer depend on
  scheduler timing.
- CLI error guidance, acceptance totals, GitHub policy visibility, task/loop
  progress, and degraded-cycle outcomes now report the state that actually
  occurred instead of optimistic or ambiguous summaries.

### Known / deferred

- The source control-plane boundary is not a hosted service. Native device login
  and credential storage ([#984](https://github.com/mlnomadpy/dacli/issues/984)),
  metadata-only project sync ([#981](https://github.com/mlnomadpy/dacli/issues/981)),
  policy/budget distribution ([#980](https://github.com/mlnomadpy/dacli/issues/980)),
  the GitHub App service ([#979](https://github.com/mlnomadpy/dacli/issues/979)),
  signed role distribution ([#978](https://github.com/mlnomadpy/dacli/issues/978)),
  and the cross-project hosted dashboard ([#983](https://github.com/mlnomadpy/dacli/issues/983))
  remain separately tracked work.
- Scheduled encrypted backups, immutable off-site retention, secret-manager
  rotation, monitored restore drills, deployment automation, production SLOs,
  and compliance certification are not claimed by this release candidate.

## [0.3.1] - 2026-08-31

### Added

- **A first-view operator pulse in the local dashboard.** The read-only
  projection now surfaces the next unfinished critical-path task, its recorded
  path, attention signals, and workspace totals before the detailed project,
  agent, burn, and role views.
- **A first-class dashboard operator guide.** The published documentation now
  explains freshness, attention semantics, responsive use, troubleshooting,
  and the dashboard's no-mutation authority boundary.

### Changed

- Reorganized the dashboard into Pulse, Delivery, Agents, and Team regions with
  keyboard-focusable sticky navigation and a shared graphite/navy visual system.
- Improved the mobile information order and allowed attention remedies to wrap
  instead of hiding their safe next action behind truncation.
- Rebuilt the GitHub Pages landing experience around the governed delivery
  lifecycle and replaced the stale dashboard image with two representative,
  fixture-backed screenshots.
- Updated the shipped Vue architecture contract and public documentation index
  so they describe the implemented dashboard rather than its historical draft.

## [0.3.0] - 2026-08-29

### Security

Hardened the CLI's own privileged surface so that routing a command **through**
`dacli` can never grant a caller more than its capability allows. The grant
model remains cooperative at the filesystem (see DESIGN § 6), but the tool no
longer acts as an escalation path.

- **Grant gate on privileged subcommands.** `shortcut add`, `runtime add`,
  `project add`, `project rm`, and `kill` — plus the remote-write levers of
  `report` (`--repo`/`--disclose`) and `escalate --github` — now refuse a
  read-only caller with exit 3 (`clikit.RequireRW`). Previously these had no
  capability check, so a read-only agent could define and run executable
  shortcuts/runtimes as the operator.
- **Path-traversal containment.** A project slug (from an explicit `--slug` or a
  forged `--project`) is validated as a single path segment (`workspace.SafeSegment`);
  a value carrying `..` or a separator can no longer read, write, or delete
  outside `.dacli`. `CreateProject` rejects such a slug up front.
- **Git option-injection closed.** Every caller-supplied ref reaches git after a
  `--` end-of-options marker (`fetch`, `push` in `gitx`; the land-status fetch
  in `vcs`), so a value like `--upload-pack=<cmd>` is treated as a refspec, never
  executed.
- **Credential env passthrough denied.** `runtime add` refuses an
  `env_passthrough` naming a known credential variable (`ANTHROPIC_API_KEY` and
  similar). Children run under the operator's own Claude Code login; the
  no-inherited-key rule is now a checked invariant rather than a default value a
  runtime edit could undo.
- **Roster wiki disclosure gate fixed.** `catalog` now probes the visibility of
  the repository it actually publishes to (via `gh repo view --repo`), instead of
  whatever the working directory's remote resolved to — so a private working repo
  can no longer publish the role/skill roster to a public wiki.

### Fixed

Correctness fixes from the same system audit. Several of these mean the tool no
longer misreports its own state.

- **The loop no longer records a zero-commit spawn as done.** A worktree/branch
  is created at spawn time, so branch existence was not evidence of work; a child
  that died before committing left an empty branch that read as an ancestor of
  trunk and was force-accepted as done. Progress is now gated on commits beyond
  trunk, and an empty branch is treated as a failed spawn.
- **CRLF files no longer lose all frontmatter.** `mdstore.Parse` normalizes line
  endings, so a Windows checkout (`core.autocrlf`) no longer yields empty
  frontmatter — every id/owner/status blank — with no error.
- **Free-text flag values can no longer corrupt a task file.** `Front.Set`
  quotes/escapes newlines and `#`, so a value like a pasted multi-line string no
  longer makes a task silently unparseable and invisible.
- **The thrash guard can fire again.** Trunk-progress excludes the loop's own
  per-cycle `.dacli` bookkeeping commit, so `--no-progress-halt` measures code
  reaching trunk rather than the loop narrating itself.
- **`loop --max-cycles N` now bounds an empty-backlog run.** An unproductive idle
  tick counts toward the bound (a productive one that files work does not), so a
  bounded run terminates instead of idling forever.
- **The loop's retro phase runs** instead of exiting with a usage error every
  cycle, and **`ship`/record-ship receive the resolved trunk** (`--into`), so the
  land phase works on repos whose trunk is not `main`.
- **`dacli wait` waits on the whole process group,** not just the leader PID, so
  the loop no longer proceeds to land while a child is mid-commit.
- **Landed task worktrees/branches are garbage-collected** on confirmed merge,
  instead of accumulating one per completed task.
- **A merge-conflict block surfaces a failed persist** instead of reporting the
  task "blocked" while it stays runnable.
- **Unknown/typo'd flags are rejected** (exit 2, naming the flag) on 51 command
  handlers, instead of being silently dropped and the command running against
  wrong or default values.

### Changed — acceptance is evidence-bound

A close now records what certified it, and verification is per task.

- **Every close records its evidence.** The task log carries either
  ``verified by `<cmd>` (exit 0)`` or `closed WITHOUT verification — no --verify
  command was given`. Previously an unverified close was indistinguishable from
  a verified one in the record, which made every `done` label an unverified
  assertion.
- **`--verify` runs per task, not once per batch.** `accept --all` used to run
  the command a single time and then close *every* proposed task — so one
  passing build closed tasks whose work was unrelated or absent. Each task is
  now verified on its own, and a task whose verification fails is skipped and
  left open.
- **New `--require-verify`** refuses (exit 3) to close anything without a
  verification command, making an unverified close impossible rather than merely
  visible. Intended for runs where the recorded trajectory is the deliverable.

### Documentation

- Added this changelog.
- DESIGN § 6 documents that the CLI does not escalate its own caller, and the two
  structural guards (path containment, git option-injection) behind it.
- Fixed the `env_passthrough` example in `docs/RUNTIMES.md`, which previously
  showed `ANTHROPIC_API_KEY` — a value the runtime now denies — and documented the
  credential denylist.
- Install docs now lead with `go install` (which works today) and mark the
  Homebrew tap and binary downloads as arriving with the first tagged release,
  so a new user is not sent to a path that 404s.
- Corrected the merged-PR count to the true figure across the landing page,
  README, and docs, and removed an unsourced "6 bugs in its own governor" claim.
- Fixed stale "not implemented / specification only" status headers on the MCP,
  SPM, TEAM, WALKTHROUGH, ARCHITECTURE, and FORMAT pages — all describe shipped
  subsystems.
- `.gitignore` now excludes the `site/` mkdocs build output.

### Known / deferred

- The `dacli` binary sits on the child agents' Bash allowlist at a writable
  path; hardening this is a deployment change (install to a non-writable
  location and allowlist that), not a code change.
- Under local development, `go build ./...` compiles a vendored `.go` file
  inside `ui/node_modules`. It is gitignored (absent in CI and clean clones),
  and the nested-module fix breaks the `ui/dist` embed, so it is left as-is.
