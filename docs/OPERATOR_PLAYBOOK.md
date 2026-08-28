# Agent orchestration playbook

dacli is the agent-facing control plane for autonomous product-building swarms.
The orchestrator agent interprets product direction, plans and judges; dacli
enforces durable state, routing, permissions, budgets, lifecycle, recovery, and
landing across replaceable coding CLIs. Humans retain credentials, exceptions,
the emergency stop, and release authority. Continuous means repeated bounded transactions with durable checkpoints, not permissionless automation. Use the
CLI help for the installed version; this playbook names shipped paths and
separates future service ideas below.

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

Start every profile by measuring the actual workspace: `dacli status`, `dacli doctor`, `dacli task list --status open --project <project>`, `dacli agents`, and `dacli loop status --project <project>`. Exit 3 is a policy answer: follow its remedy; do not retry unchanged.

## GitHub-first critical-path cycle

1. Link and inspect the repository with `dacli github link <project>` (`--allow-public` is required after deliberately reviewing disclosure for a public repository), then run `dacli github doctor`. List existing open and active tasks, inspect open issues in GitHub, and preview inbound adoption with `dacli github pull <project> --dry-run` (`github sync <project> --dry-run` previews both inbound and outbound halves). Compare proposed issue titles with existing work before the real pull because issue adoption prevents duplicate issue mappings but does not perform semantic deduplication. Before adoption, make each issue's `## Acceptance criteria` checkbox list independently checkable; the shipped CLI has no task-edit command that can add missing criteria after pull.
2. `dacli github pull <project>` adopts human issues. Estimate the resulting tasks with `dacli task estimate <ref> --estimate o,m,p`. Add dependency edges at creation with `--depends-on ref[:TYPE]`, or refine any existing/adopted task with `dacli task depend <ref> --add dep[:FS|SS|FF|SF] --remove dep[:FS|SS|FF|SF]`. References may be project-qualified (`project/ref`); validation refuses missing, ambiguous, self, and cyclic edges before writing. Non-owners record a proposal for the owner's `dacli sync`, preserving the adopted task's GitHub mapping. While the recorded graph is incomplete, do not use `critical-path` or `next` to claim that incomplete graph is authoritative. Run `dacli critical-path --project <project>` and `dacli next --project <project> --parallel <width>` only after the recorded graph is truthful; protect zero-slack work and avoid spending WIP on nonblocking work.
3. Resolve the harness policy first (`--harness <family>` for one CLI; repeated values plus `--hybrid` for an explicit cross-harness allowlist). Then route models inside that boundary with `dacli team assign <ref>` and `dacli preflight --role <role>`; preview a paid run with `dacli spawn --task <ref> --role <role> --harness <family> --advise --max-tokens N`. Launch a writer in a narrow worktree claim only when the preview says the runtime ceiling is `ENFORCED`. `UNSUPPORTED` refuses at launch; `--allow-advisory-tokens` is an explicit accounting downgrade, not a cap.
4. Observe `dacli agents --tail` and `dacli logs <run-id-prefix|child-id> -f`, then `dacli wait` and `dacli sync`. After a restart, inspect `dacli runs list`, the relevant `runs show`, loop status, PR state, and trunk before resuming. Verify the repository's own bar; use `verify` with a diverse panel when consequence warrants it. The loop's `--review-role` runs after landing to find the next improvement; it is not a pre-merge security gate. Add an independent pre-merge review or required GitHub check when consequence warrants it. The owner checks acceptance only after evidence exists.
5. Choose and record project landing policy before execution: `dacli project show <slug> --landing-mode pr --landing-base main` persists PR landing and its base. For a direct task, push its branch with `dacli push <ref>`, then open a PR with `dacli pr --task <ref> --with-verdicts`; add `--auto` only when protected, trustworthy required checks and review policy make unattended merging safe. In a PR-mode loop, the controller repeats those remote steps after a worker commits: it pushes the canonical task branch and creates or reuses its PR against the effective base, so a worker exit before its prompted PR step does not strand the commit. To finish an explicitly selected checked task without the accept-before-merge cycle, run `dacli ship --project <slug> --tasks <ref> --pr --verify "<cmd>"`. It remains the separate governed wave transaction that accepts and integrates the selected work: it leaves the task nonterminal through the checks-gated merge, inspects fresh configured trunk, verifies there, accepts, and only then closes the mapped issue through least-disclosure `github push --closure-only`; findings and decisions are not published by that completion step. Before owner acceptance or GitHub issue closure, the transaction must observe both the merged PR and its commit on trunk. Use `dacli pr status --task <ref>` to ask whether it landed and `dacli pr diagnose --task <ref> --json` to classify CI, access, approval, or topology blockers. Record `dacli retro <task-or-project-ref> --well "..." --bad "..." --improve "..."` after landing.
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

For command-level detail, read the skill references for [operating profiles](../skills/dacli/references/operating-profiles.md), [model economics](../skills/dacli/references/model-economics.md), [critical-path GitHub work](../skills/dacli/references/critical-path-github.md), and [continuous operations](../skills/dacli/references/continuous-operations.md).

## Workspace boundaries that preserve collaboration

Projects isolate task lists, schedules, goals, and backlog views, so tasks do not leak into another project's normal work queue. Direct task references are workspace-wide: ambiguous short references fail, while project-qualified shorthand and task ULIDs make the target explicit. The workspace-wide append-only record deliberately shares agents, events, notes, runtimes, skills, runs, and findings. A linked worktree keeps code and branch writes isolated while reporting identity/events to that one workspace record. Use project-qualified references for cross-project dependency edges, and let the GitHub mapping associate each local task with its mirrored issue. GitHub is the primary shared collaboration surface for orchestrator agents, coding agents, and humans; dacli's local record is the canonical execution and evidence ledger.

## Shipped, experimental, and future

**Shipped local behavior:** projects/tasks/dependencies, critical-path and next selection, roles/runtimes, worktrees/claims, bounded loops, persisted operating profiles, a single-project service supervisor, governor/landing/service journals, PR/issue mirroring, runtime cooldowns, leases, circuit breakers, and manual STOP files. Service is many finite loop subprocesses, never one infinite loop. No dedicated runtime-cooldown clear or expiry command is shipped; diagnose the recorded condition instead of documenting an invented reset.

**Experimental or authority-configured behavior:** vendor adapters, provider fallback chains, auto-merge availability, and GitHub project synchronization. Probe with `runtime doctor`, `preflight`, and `github doctor`; do not infer that a provider's flags, quota, or GitHub setting is healthy.

Run bounded landing loops for multiple projects sharing one repository/trunk sequentially unless repository policy explicitly proves concurrent integration safe; project isolation does not create separate Git histories.

**Future service/SaaS/GitHub-App vision:** the shipped service profile is a local, single-project supervisor, not a multi-tenant control plane. Organization policy distribution and loop-integrated dead-letter recovery remain future work. Running it on a VPS grants no broader authority. The private GitHub App bridge is an optional, deliberately constrained event/check adapter; it does not grant permission to publish releases or replace the local record. Release publication is a separate default-off policy and choosing `service` never enables it.
