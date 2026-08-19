# Cost-aware critical-path operator playbook

dacli is a policy-driven continuous engineering controller over replaceable coding CLIs. It is not permissionless automation: Continuous means repeated bounded transactions with durable checkpoints. Use the CLI help for the installed version; this playbook names the shipped command paths and separates future ideas below.

## Choose the smallest operating profile

| Need | First choice | Boundary |
|---|---|---|
| Understand a repository or backlog | Inspect: `overview`, `status`, `doctor`, `task list` | Read only; do not create busywork. |
| Finish one contained change | Single task: estimate, assign, context, spawn/worktree, verify | One narrow claim and one PR. |
| Deliver several independent changes | Supervised wave: `next --parallel`, detached worktrees, `wait`, `sync`, `ship` | Fixed WIP and explicit landing window. |
| Improve a backlog repeatedly | Bounded loop: `loop --max-cycles` plus budgets and idle halt | Checkpoints make it resumable, not infinite. |
| Centralize multi-repo/always-on control | Future service/control plane | Not a shipped local deployment model. |

Start every profile by measuring the actual workspace: `dacli status`, `dacli doctor`, `dacli task list --status open --project <project>`, `dacli agents`, and `dacli loop status --project <project>`. Exit 3 is a policy answer: follow its remedy; do not retry unchanged.

## GitHub-first critical-path cycle

1. Link and inspect the repository with `dacli github link <project>` (`--allow-public` is required after deliberately reviewing disclosure for a public repository), then run `dacli github doctor`. List existing open and active tasks, inspect open issues in GitHub, and preview inbound adoption with `dacli github pull <project> --dry-run` (`github sync <project> --dry-run` previews both inbound and outbound halves). Compare proposed issue titles with existing work before the real pull because issue adoption prevents duplicate issue mappings but does not perform semantic deduplication. Before adoption, make each issue's `## Acceptance criteria` checkbox list independently checkable; the shipped CLI has no task-edit command that can add missing criteria after pull.
2. `dacli github pull <project>` adopts human issues. Give each task checkable acceptance criteria and an estimate with `dacli task estimate <ref> --estimate o,m,p`. Add dependency edges when creating local tasks (`--depends-on ref[:TYPE]`); the shipped CLI cannot add an edge to an already-adopted task, so stop and reconcile that backlog rather than inventing a critical path. Run `dacli critical-path --project <project>` and `dacli next --project <project> --parallel <width>` only after the recorded graph is truthful; protect zero-slack work and avoid spending WIP on nonblocking work.
3. Route with `dacli team assign <ref>` and `dacli preflight --role <role>`; preview a paid run with `dacli spawn --task <ref> --role <role> --advise`. Then launch a writer in a narrow worktree claim, bounded by `--max-tokens`.
4. Observe `dacli agents --tail` and `dacli logs <run-id-prefix|child-id> -f`, then `dacli wait` and `dacli sync`. After a restart, inspect `dacli runs list`, the relevant `runs show`, loop status, PR state, and trunk before resuming. Verify the repository's own bar; use `verify` with a diverse panel when consequence warrants it. The loop's `--review-role` runs after landing to find the next improvement; it is not a pre-merge security gate. Add an independent pre-merge review or required GitHub check when consequence warrants it. The owner checks acceptance only after evidence exists.
5. Choose and record project landing policy before execution: `dacli project show <slug> --landing-mode pr --landing-base main` persists PR landing and its base. For a direct task, push its branch with `dacli push <ref>`, then open a PR with `dacli pr --task <ref> --with-verdicts`; add `--auto` only when protected, trustworthy required checks and review policy make unattended merging safe. Before owner acceptance or GitHub issue closure, use `dacli pr status --task <ref>` and fetch/inspect the target branch to observe both the merged PR and its commit on trunk. Use `ship` for the separate governed wave transaction that accepts and integrates the reviewed task window; it is not a prerequisite for a direct task PR. Only after the observed landing should the owner accept, synchronize/close the issue, and record `dacli retro <task-or-project-ref> --well "..." --bad "..." --improve "..."`.
6. Re-run critical path, calibrate from observed usage, and repeat only while evidence-backed ready work remains. `github push <project> --dry-run` and `github sync --dry-run` preview projection changes first.

For command-level detail, read the skill references for [operating profiles](../skills/dacli/references/operating-profiles.md), [model economics](../skills/dacli/references/model-economics.md), [critical-path GitHub work](../skills/dacli/references/critical-path-github.md), and [continuous operations](../skills/dacli/references/continuous-operations.md).

## Workspace boundaries that preserve collaboration

Projects isolate task lists, schedules, goals, and backlog views, so tasks do not leak into another project's normal work queue. Direct task references are workspace-wide: ambiguous short references fail, while project-qualified shorthand and task ULIDs make the target explicit. The workspace-wide append-only record deliberately shares agents, events, notes, runtimes, skills, runs, and findings. A linked worktree keeps code and branch writes isolated while reporting identity/events to that one workspace record. Use project-qualified references for cross-project dependency edges, and let the GitHub mapping associate each local task with its mirrored issue. GitHub is the primary human collaboration surface; dacli's local record is the execution and evidence ledger.

## Shipped, experimental, and future

**Shipped local behavior:** projects/tasks/dependencies, critical-path and next selection, roles/runtimes, worktrees/claims, bounded loops, persisted governor and landing journals, PR/issue mirroring, runtime cooldowns, and manual STOP files. No dedicated runtime-cooldown clear or expiry command is shipped; diagnose the recorded condition instead of documenting an invented reset.

**Experimental or operator-configured behavior:** vendor adapters, provider fallback chains, auto-merge availability, and GitHub project synchronization. Probe with `runtime doctor`, `preflight`, and `github doctor`; do not infer that a provider's flags, quota, or GitHub setting is healthy.

Run bounded landing loops for multiple projects sharing one repository/trunk sequentially unless repository policy explicitly proves concurrent integration safe; project isolation does not create separate Git histories.

**Future service/SaaS/GitHub-App vision:** an always-on control plane or multi-tenant service is not implied by `loop`. Service-level circuit breakers, cooldown policies, and dead-letter queues are design requirements, not shipped operator commands. The private GitHub App bridge is an optional, deliberately constrained event/check adapter; it does not grant permission to publish releases or replace the local record. Release publication is default-off and requires explicit human authority.
