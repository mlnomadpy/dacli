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

1. Link and inspect the repository, then preview inbound work: `dacli github doctor`; `dacli github pull <project>` adopts human issues. Deduplicate with `dacli task list --status open --project <project>` before creating a task.
2. Give each task checkable acceptance criteria, an estimate with `dacli task estimate <ref> --estimate o,m,p`, and dependency edges at creation (`--depends-on ref[:TYPE]`). Run `dacli critical-path --project <project>` and `dacli next --project <project> --parallel <width>`; protect zero-slack work and avoid spending WIP on nonblocking work.
3. Route with `dacli team assign <ref>` and `dacli preflight --role <role>`; preview a paid run with `dacli spawn --task <ref> --role <role> --advise`. Then launch a writer in a narrow worktree claim, bounded by `--max-tokens`.
4. Observe `agents --tail` and `logs -f`, then `wait` and `sync`. Verify the repository's own bar; use `verify` with a diverse panel when consequence warrants it. The owner checks acceptance only after evidence exists.
5. Push the task branch with `dacli push <ref>`, then open its PR with `dacli pr --task <ref> --with-verdicts --auto`. Confirm checks and merged state (`dacli pr status --task <ref>`); then use the configured landing policy, sync/close the task and issue, and record `dacli retro <ref> --well "..." --bad "..." --improve "..."`.
6. Re-run critical path, calibrate from observed usage, and repeat only while evidence-backed ready work remains. `github push <project> --dry-run` and `github sync --dry-run` preview projection changes first.

For command-level detail, read the skill references for [operating profiles](../skills/dacli/references/operating-profiles.md), [model economics](../skills/dacli/references/model-economics.md), [critical-path GitHub work](../skills/dacli/references/critical-path-github.md), and [continuous operations](../skills/dacli/references/continuous-operations.md).

## Workspace boundaries that preserve collaboration

Projects isolate task lists, schedules, goals, and backlog views, so tasks do not leak into another project's normal work queue. Direct task references are workspace-wide: ambiguous short references fail, while project-qualified shorthand and task ULIDs make the target explicit. The workspace-wide append-only record deliberately shares agents, events, notes, runtimes, skills, runs, and findings. A linked worktree keeps code and branch writes isolated while reporting identity/events to that one workspace record. Use project-qualified references for cross-project dependency edges, and let the GitHub mapping associate each local task with its mirrored issue. GitHub is the primary human collaboration surface; dacli's local record is the execution and evidence ledger.

## Shipped, experimental, and future

**Shipped local behavior:** projects/tasks/dependencies, critical-path and next selection, roles/runtimes, worktrees/claims, bounded loops, persisted governor and landing journals, PR/issue mirroring, runtime cooldowns, and manual STOP files.

**Experimental or operator-configured behavior:** vendor adapters, provider fallback chains, auto-merge availability, and GitHub project synchronization. Probe with `runtime doctor`, `preflight`, and `github doctor`; do not infer that a provider's flags, quota, or GitHub setting is healthy.

**Future service/SaaS/GitHub-App vision:** an always-on control plane or multi-tenant service is not implied by `loop`. The private GitHub App bridge is an optional, deliberately constrained event/check adapter; it does not grant permission to publish releases or replace the local record. Release publication is default-off and requires explicit human authority.
