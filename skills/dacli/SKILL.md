---
name: dacli
description: "Run autonomous product-building swarms with dacli. Use when an orchestrator AI agent must plan a critical-path backlog, route models across coding-agent CLIs, execute bounded task/wave/loop profiles, recover durable state, or verify and land work through GitHub."
---

# dacli

Act as the orchestrator agent. Use dacli as the agent-facing control plane over
replaceable coding CLIs: interpret the product direction, maintain the critical
path, choose capable and economical roles/models, and judge evidence while
dacli enforces state, permissions, budgets, lifecycle, recovery, and landing.
Continuous operation is repeated bounded transactions with durable checkpoints,
not permissionless execution.

Keep authority explicit. The human governor supplies direction, credentials,
exceptions, emergency stop, and release policy. Worker coding agents implement,
review, and test. GitHub is the shared collaboration and landing surface; the
local dacli workspace is the canonical execution and evidence ledger.

Repository `AGENTS.md` and `CONTRIBUTING.md` override this skill. Measure live
state before acting:

```bash
dacli status
dacli doctor
dacli task list --status open --project <project>
dacli agents --tail
dacli loop status --project <project>
```

## Choose the first mode

| Situation | First choice |
|---|---|
| Understand work | `dacli start --profile inspect` |
| One bounded change | `dacli start --profile task` |
| Independent ready tasks | `dacli start --profile wave` with fixed WIP and claims |
| Repeatable backlog improvement | `dacli start --profile loop` with bounded cycles and budgets |
| Resident single-project runner | `dacli start --profile service`; repeated finite loops |
| Multi-repo or multi-tenant control | Future control-plane vision, not a local-service promise |

Preview the resolved project policy before execution:

```bash
dacli start --project <project> --profile <mode> --dry-run
dacli start --project <project> --profile <mode> --configure
dacli start --project <project> --show --json
```

`--dry-run` neither writes nor launches; `--configure` persists without
launching. Inspect execution is read-only. Every other profile persists its
resolution and then delegates to the existing bounded loop strategy. Service
adds a lease and durable checkpoints around repeated finite invocations; it
never grants tag or release authority.

Never manufacture backlog work. Exit 3 is a policy refusal: stop and follow
the stated remedy rather than retrying unchanged.

## Read one focused reference before acting

- [operating-profiles.md](references/operating-profiles.md): choose task,
  wave, or bounded loop; WIP and budgets.
- [model-economics.md](references/model-economics.md): capability, Te, context,
  consequence uplift, quota/health, and independent review diversity.
- [critical-path-github.md](references/critical-path-github.md): deduplicate,
  estimate/depend, critical path, GitHub issue/PR/check/merge cycle.
- [continuous-operations.md](references/continuous-operations.md): STOP,
  heartbeats, journals, breakers, dead letters, observability, recovery.
- [workspace-tasks-projects.md](references/workspace-tasks-projects.md):
  project isolation, shared workspace state, references, and GitHub mappings.
- [runtimes-models-skills.md](references/runtimes-models-skills.md): runtime
  setup and symmetric Codex/Claude Code/Gemini/Copilot/generic adapters.
- [roster-design.md](references/roster-design.md): capability tiers, role sizing,
  provider diversity, and reusable team policy.
- [swarms-loops.md](references/swarms-loops.md): claims, worktrees, supervision,
  review timing, and bounded loop execution.
- [recovery.md](references/recovery.md): refusals, partial runs, PR conflicts,
  journals, and safe restart inspection.
- [github-landing.md](references/github-landing.md): GitHub disclosure,
  PR/landing policy, and release boundary.

The canonical orchestration narrative is
[`docs/OPERATOR_PLAYBOOK.md`](../../docs/OPERATOR_PLAYBOOK.md) in the source
repository. Keep this file a router; do not duplicate its detailed runbooks.
