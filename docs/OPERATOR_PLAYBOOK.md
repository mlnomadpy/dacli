# Agent orchestration playbook

dacli is the agent-facing control plane for autonomous product-building swarms.
The orchestrator agent interprets product direction, plans and judges; dacli
enforces durable state, routing, permissions, budgets, lifecycle, recovery, and
landing across replaceable coding CLIs. Humans retain credentials, exceptions,
the emergency stop, and release authority. Continuous means repeated bounded transactions with durable checkpoints, not permissionless automation. Use the
CLI help for the installed version; this playbook names shipped paths and
separates future service ideas below.

## Bootstrap an agent session

Treat the skill and binary as a negotiated pair, then read sourced JSON state
before choosing work:

```bash
dacli version --compatibility --json
dacli capabilities --json
dacli whoami --json
dacli status --project <project> --json
dacli task list --status open --project <project> --json
dacli next --project <project> --critical-path --json
dacli agents --project <project> --json
dacli loop status --project <project> --json
dacli explain --project <project> --json
```

Required compatibility gaps are policy refusals, not warnings to ignore. Human
text is for operators; agent branching should use advertised JSON schemas and
typed exit codes. The v0.3.0 release is the first published binary baseline,
but capabilities—not semantic-version guesses—decide whether installed
guidance is executable.

## Choose the smallest operating profile

| Need | First choice | Boundary |
|---|---|---|
| Understand a repository or backlog | Inspect: `overview`, `status`, `doctor`, `task list` | Read only; do not create busywork. |
| Finish one contained change | Single task: estimate, assign, context, spawn/worktree, verify | One narrow claim and one PR. |
| Deliver several independent changes | Supervised wave: `next --parallel`, detached worktrees, `wait`, `sync`, `ship` | Fixed WIP and explicit landing window. |
| Improve a backlog repeatedly | Bounded loop: `loop --max-cycles` plus budgets and idle halt | Checkpoints make it resumable, not infinite. |
| Keep one project improving on a resident runner | `start --profile service` | A leased supervisor repeatedly invokes finite loops and stops at durable breaker/budget/landing checkpoints. |

`dacli start --project <slug> --profile <name> --dry-run` resolves the complete policy and previews selected tasks, CPM slack, claims, harness boundary, model routing, budgets, verification, landing, release, and recovery without writing or launching work. A saved project profile is the base: `--profile` and `--width` replace only those explicit fields, never its verification or landing policy. First-time configuration derives verification commands from the adopted codebase map and refuses an unknown stack instead of guessing Go; refresh with `dacli adopt --project <slug>` when detection is stale. Auto-merge defaults off and is forwarded explicitly to the loop; enable it only after repository checks/reviews make it safe. Drop `--dry-run` to persist the policy and execute it; add `--configure` to persist without execution. `dacli start --project <slug> --show --json` reads the persisted project policy for automation. With no `--profile`, `start` prompts for the same five choices.

Select the coding harness independently of model cost. `--harness codex` pins
implementation, review, recovery, and fallback to Codex-family adapters while
still allowing cost/capability routing among Codex models. A different model or
review role does not imply permission to switch CLIs. Cross-harness routing is
explicit: repeat `--harness` and add `--hybrid`, for example `--harness codex
--harness claude --hybrid`. Preflight every allowed harness; hybrid is an
allowlist, not a health claim.

For a manual wave, width limits concurrency but not aggregate spend. Preview
each launch with `spawn --advise`, accept only an `ENFORCED` token ceiling, and
allocate per-spawn `--max-tokens` values whose sum fits the wave budget. Use a
bounded loop with `--window-tokens`/`--token-window` when the required contract
is a durable rolling total.

Start every profile by measuring the actual workspace: `dacli status --project <project>`, `dacli doctor`, `dacli task list --status open --project <project>`, `dacli agents --project <project> --json`, and `dacli loop status --project <project>`. During supervised operation, the dashboard Agents route combines those durable loop, recovery, phase, reservation, routing, outcome-window, and worker records for one selected project; it never resumes a loop or refreshes GitHub. Treat outcome comparisons as descriptive: inspect their sample, coverage, task size, exact evidence membership, and unknown fields before changing a route or model. Provider-reported USD is not billing, and a missing cost is not zero. Use `dacli next --project <project> --critical-path` for an explicit critical-path selection and `dacli route <path>` for the same ownership answer as `team route`. Exit 3 is a policy answer: follow its remedy; do not retry unchanged.

Use the dashboard Overview attention queue to triage, not to govern. Its
`operator-attention/v1` projection ranks canonical token, no-progress,
capacity/WIP, critical-path, verification, CI/review/billing, recovery, and
owner-handoff conditions. Open the exact linked task, run, check, or dependency
evidence and take the printed next safe action in the CLI or GitHub. There is no
dashboard dismissal: if an item is wrong, repair or re-observe its source
record; if its freshness is stale or external state is unknown, do not infer
resolution.

## GitHub-first critical-path cycle

1. Link and inspect the repository with `dacli github link <project>` (`--allow-public` records only the public-safe allowlist; `--allow-internal` is a separate exact-repository decision), then run `dacli github doctor` and `dacli github projection <project> --json`. List existing open and active tasks, inspect open issues in GitHub, and preview inbound adoption with `dacli github pull <project> --dry-run` (`github sync <project> --dry-run` previews both inbound and outbound halves). Compare proposed issue titles with existing work before the real pull because issue adoption prevents duplicate issue mappings but does not perform semantic deduplication. Before adoption, make each issue's `## Acceptance criteria` checkbox list independently checkable. For an already-adopted task, preview a content-addressed correction with `dacli task acceptance migrate <ref> --dry-run`, then apply only the exact current plan with `dacli task acceptance migrate <ref> --apply <plan-id>`; ambiguous or changed GitHub text refuses without rewriting existing local checks.
2. `dacli github pull <project>` adopts human issues. Estimate the resulting tasks with `dacli task estimate <ref> --estimate o,m,p`. Add dependency edges at creation with `--depends-on ref[:TYPE]`, or refine any existing/adopted task with `dacli task depend <ref> --add dep[:FS|SS|FF|SF] --remove dep[:FS|SS|FF|SF]`. References may be project-qualified (`project/ref`); validation refuses missing, ambiguous, self, and cyclic edges before writing. Non-owners record a proposal for the owner's `dacli sync`, preserving the adopted task's GitHub mapping. While the recorded graph is incomplete, do not use `critical-path` or `next` to claim that incomplete graph is authoritative. Run `dacli critical-path --project <project>` and `dacli next --project <project> --parallel <width>` only after the recorded graph is truthful; protect zero-slack work and avoid spending WIP on nonblocking work.
3. Resolve the harness policy first (`--harness <family>` for one CLI; repeated values plus `--hybrid` for an explicit cross-harness allowlist). Then route models inside that boundary with `dacli team assign <ref>` and `dacli preflight --role <role>`; preview a paid run with `dacli spawn --task <ref> --role <role> --harness <family> --advise --max-tokens N`. Launch a writer in a narrow worktree claim only when the preview says the runtime ceiling is `ENFORCED`. `UNSUPPORTED` refuses at launch; `--allow-advisory-tokens` is an explicit accounting downgrade, not a cap.
4. Observe the selected project in the dashboard Agents route, or use `dacli agents --tail` and `dacli logs <run-id-prefix|child-id> -f`, then `dacli wait` and `dacli sync`. After a restart, inspect `dacli runs list`, the relevant `runs show`, loop status, the persisted token reservations and phase checkpoint, PR state, and trunk before resuming. A dashboard state of stale, partial, corrupt, external unknown, or halted policy is evidence to investigate—not launch authority. RW providers execute in an independent `claim-sandbox/v1` checkout; only exact claimed regular-file additions/modifications project into the clean canonical worktree. An out-of-claim/generated write, traversal, symlink, rename, delete, stale head, or dirty base refuses without touching canonical code. If wider work is genuinely required, root records `claim expand --task <ref> --run <terminal-run> --add <path> --reason <why>` before relaunch; never derive authority from attempted writes. A worker whose sandbox cannot write the linked worktree's shared Git index is finalized through `parent-commit-request-v1.json` and `parent-commit-receipt-v1.json`; the parent re-observes the exact claim/content/tree and the loop resumes from its durable `committed` phase. Verify the repository's own bar; use `verify` with a diverse panel when consequence warrants it. The governed delivery review runs before landing when configured; its preflight emits an exact `runtime-launch-contract/v1` fingerprint and the loop supplies that fingerprint to the later structured-review spawn. Inspect `launch-contract.json` and `review-validation.json` when it refuses: cooperative RO, RW, changed runtime/model/sandbox, or stale commit/tree cannot become independent approval. The loop's later improvement review remains distinct. Add a required GitHub check when consequence warrants it. The owner checks acceptance only after exact-tree evidence exists.
5. Choose and record project landing policy before execution: `dacli project show <slug> --landing-mode pr --landing-base main` persists PR landing and its base. For a direct task, run `dacli pr --task <ref> --with-verdicts`: it publishes the exact canonical task branch itself before creating or reusing the PR, and checkpoints the branch/commit/PR identity for restart. A separate `dacli push <ref>` is only a deliberate pre-publication or divergence-recovery step. Add `--auto` only when protected, trustworthy required checks and review policy make unattended merging safe. Use `dacli pr wait --task <ref> --timeout <seconds> --json` for a bounded wait on the same typed diagnosis, and `dacli pr land --task <ref> --base main --merge` as the task-level delegation to the canonical integration transaction. In a PR-mode loop, the controller repeats those remote steps after a worker commits: it pushes the canonical task branch and creates or reuses its PR against the effective base, so a worker exit before its prompted PR step does not strand the commit. To finish an explicitly selected checked task without the accept-before-merge cycle, run `dacli ship --project <slug> --tasks <ref> --pr --verify "<cmd>"`. It remains the separate governed wave transaction that accepts and integrates the selected work: it leaves the task nonterminal through the checks-gated merge, inspects fresh configured trunk, verifies there, accepts, and only then closes the mapped issue through least-disclosure `github push --closure-only`; findings and decisions are not published by that completion step. Before owner acceptance or GitHub issue closure, the transaction must observe both the merged PR and its commit on trunk. Use `dacli pr status --task <ref>` to ask whether it landed and `dacli pr diagnose --task <ref> --json` to classify CI, access, approval, or topology blockers. Audit cleanup with `dacli branches audit --project <slug> --json` and pass only that exact plan id to `branches prune`; both delegate to cleanup and never invent deletion authority. Record `dacli retro <task-or-project-ref> --well "..." --bad "..." --improve "..."` after landing.

In PR mode, `task done` and a synced child completion proposal never mean
landed. They persist the nonterminal `completion_state: implemented-unlanded`
request and return exit 3 with the landing/acceptance remedy. Only the
acceptance owner clears that state after the configured base,
exact reviewed head/tree, and required external checks/artifacts are observed.
Local landing mode retains the bounded direct completion path.
6. Re-run critical path, calibrate from observed usage, and repeat only while evidence-backed ready work remains. `github push <project> --dry-run` and `github sync <project> --dry-run` preview projection changes first.

After a completed wave, use `dacli cleanup --project <slug> --dry-run` to
classify managed worktrees, branches, task/run claims, PR history, and generated
run artifacts together. Apply only the exact reviewed identity with `dacli
cleanup --project <slug> --apply-safe <plan-id>`; changed or unreadable evidence
must produce a new plan or a refusal, never an inferred deletion. Eligible
generated `*.tmp` run artifacts are moved individually into the plan-keyed
workspace quarantine; durable process records, outcomes, transcripts, and
verification evidence remain in place. Recover an audited artifact without
overwriting its source with
`dacli cleanup --project <slug> --restore <plan-id> --artifact <identity>`.

For command-level detail, read the repository skill references for [operating profiles](https://github.com/mlnomadpy/dacli/blob/main/skills/dacli/references/operating-profiles.md), [model economics](https://github.com/mlnomadpy/dacli/blob/main/skills/dacli/references/model-economics.md), [critical-path GitHub work](https://github.com/mlnomadpy/dacli/blob/main/skills/dacli/references/critical-path-github.md), and [continuous operations](https://github.com/mlnomadpy/dacli/blob/main/skills/dacli/references/continuous-operations.md).

## Workspace boundaries that preserve collaboration

Projects isolate task lists, schedules, goals, and backlog views, so tasks do not leak into another project's normal work queue. Direct task references are workspace-wide: ambiguous short references fail, while project-qualified shorthand and task ULIDs make the target explicit. The workspace-wide append-only record deliberately shares agents, events, notes, runtimes, skills, runs, and findings. A linked worktree keeps code and branch writes isolated while reporting identity/events to that one workspace record. Use project-qualified references for cross-project dependency edges, and let the GitHub mapping associate each local task with its mirrored issue. GitHub is the primary shared collaboration surface for orchestrator agents, coding agents, and humans; dacli's local record is the canonical execution and evidence ledger.

## Shipped, experimental, and future

**Shipped local behavior:** projects/tasks/dependencies, critical-path and next selection, roles/runtimes, worktrees/claims, bounded loops, persisted operating profiles, a single-project service supervisor, governor/landing/service journals, PR/issue mirroring, runtime cooldowns, leases, circuit breakers, and manual STOP files. Service is many finite loop subprocesses, never one infinite loop. No dedicated runtime-cooldown clear or expiry command is shipped; diagnose the recorded condition instead of documenting an invented reset.

**Experimental or authority-configured behavior:** vendor adapters, provider fallback chains, auto-merge availability, and GitHub project synchronization. Probe with `runtime doctor`, `preflight`, and `github doctor`; do not infer that a provider's flags, quota, or GitHub setting is healthy.

Run bounded landing loops for multiple projects sharing one repository/trunk sequentially unless repository policy explicitly proves concurrent integration safe; project isolation does not create separate Git histories.

**Future service/SaaS/GitHub-App vision:** the shipped service profile is a local, single-project supervisor, not a multi-tenant control plane. Organization policy distribution and loop-integrated dead-letter recovery remain future work. Running it on a VPS grants no broader authority. The private GitHub App bridge is an optional, deliberately constrained event/check adapter; it does not grant permission to publish releases or replace the local record. Release publication is a separate default-off policy and choosing `service` never enables it.
