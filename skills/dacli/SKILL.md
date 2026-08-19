---
name: dacli
description: "Operate repositories with dacli as a governed, cost-aware controller for coding-agent CLIs: plan a critical-path backlog, route roles and models, run bounded waves or loops, land GitHub work, and recover durable state."
---

# dacli

Use dacli when work benefits from durable coordination, explicit policy, a
shared record, or more than one agent. It is a controller over replaceable
coding CLIs, not a permissionless autonomous service: continuous operation is
repeated bounded transactions with durable checkpoints.

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
| Understand work | Inspect: `overview`, `status`, `doctor`, task list |
| One bounded change | Estimate → assign → worktree task → verify → PR |
| Independent ready tasks | Supervised wave with fixed WIP and claims |
| Repeatable backlog improvement | Bounded `loop --max-cycles` with budgets/idle halt |
| Always-on/multi-repo control | Future service vision, not a local-loop promise |

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
- [github-landing.md](references/github-landing.md): GitHub disclosure,
  PR/landing policy, and release boundary.

The canonical operator narrative is
[`docs/OPERATOR_PLAYBOOK.md`](../../docs/OPERATOR_PLAYBOOK.md) in the source
repository. Keep this file a router; do not duplicate its detailed runbooks.
