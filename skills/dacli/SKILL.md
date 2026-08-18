---
name: dacli
description: Operate repositories with dacli as a governed team of coding-agent CLIs. Use when Codex must plan or audit a backlog, coordinate parallel agents or worktrees, select roles/models/runtimes, run a bounded dacli loop or swarm, synchronize GitHub issues and PRs, verify and land agent work, recover failed runs, or explain dacli workspace, project, task, runtime, skill, permission, and cost semantics.
---

# dacli

Use `dacli` as the coordination and evidence layer for work that is larger than
one sitting, benefits from parallel agents, or needs a durable record. Skip it
for a trivial isolated edit when the coordination overhead adds no value.

Treat `.dacli/` as the workspace record: tasks, roles, runtimes, briefs, events,
runs, decisions, findings, and verification. Treat coding-agent CLIs as
replaceable execution engines behind runtime adapters.

## Read the relevant reference

Read only the references needed for the request:

- Read [workspace-tasks-projects.md](references/workspace-tasks-projects.md)
  before creating projects or tasks, resolving task references, changing
  status, or reasoning about project isolation and cross-project dependencies.
- Read [runtimes-models-skills.md](references/runtimes-models-skills.md) before
  adding or choosing Codex, Claude Code, Gemini, Copilot, or generic runtimes;
  editing roles; routing models; setting budgets; or compiling skills.
- Read [roster-design.md](references/roster-design.md) when provisioning or
  auditing a team, removing provider/task-specific roles, choosing model tiers,
  assigning reusable skills, or setting cross-runtime review policy.
- Read [swarms-loops.md](references/swarms-loops.md) before spawning parallel
  agents, using worktrees or claims, supervising, verifying, or running a loop.
- Read [github-landing.md](references/github-landing.md) before creating or
  synchronizing GitHub issues, opening or reviewing PRs, integrating, shipping,
  releasing, or changing branch policy.
- Read [recovery.md](references/recovery.md) when a command refuses, an agent
  stalls, CI fails, a PR conflicts, a worktree accumulates, records disagree,
  or a loop stops making progress.

Repository-specific `AGENTS.md` and `CONTRIBUTING.md` instructions override this
general playbook. Read them before mutating a repository.

## Start by measuring the workspace

Run the tool; do not infer its state from filenames or a previous transcript.
If `dacli` is not installed while working on dacli itself, use
`go run ./cmd/dacli` in place of `dacli`.

```bash
dacli whoami
dacli version
dacli project list
dacli status
dacli overview
dacli doctor
dacli task list --status open --project <project>
dacli agents
dacli loop status --project <project>
git status --short --branch
```

Establish the current project, branch, acting grant, dirty paths, live agents,
open work, and integrity findings before choosing an action. Do not overwrite
or stage another agent's changes.

## Choose the operating mode

Use the smallest mode that fits the work:

| Situation | Mode |
|---|---|
| One small, local change | Work directly; keep normal repository verification |
| One delegated task | `context` → `team assign` → `spawn` → `wait` → verify |
| Several independent tasks | One claimed worktree per task; run a detached wave |
| A difficult task needing correction turns | `supervise --max-turns N` |
| A backlog to drain repeatedly | Bounded `loop --project ...` |
| Audit or review | Read-only role; findings/events; independent verifier |

Do not manufacture backlog work to justify a loop. An evidence-backed empty
cycle is a valid result.

## Create work that another agent can finish

Check for duplicates first, then create one-sitting tasks on the selected
project. Lead titles with the intent-bearing verb because routing uses it.

```bash
dacli task list --status open --project <project>
dacli task add "Fix <observable defect>" --project <project> \
  --accept "<command or inspection proves behavior A>" \
  --accept "<command or inspection proves behavior B>"
dacli task estimate <ref> --estimate <optimistic>,<probable>,<pessimistic>
dacli lint --project <project>
```

Make acceptance criteria independently checkable. State the file, command, and
observable result. Decompose work that exceeds every role's capacity rather
than merely assigning a larger model.

## Route by capability, cost, and consequence

```bash
dacli runtime doctor
dacli preflight --role <role>
dacli team assign <ref>
dacli context <ref>
```

Treat `team assign` as a cost-aware floor, not the whole decision. Raise the
role/model when blast radius, ambiguity, security, data loss, or architectural
judgment is high even if the edit is small. Confirm all of these before spawn:

1. The role kind is allowed in the project's current phase.
2. The role's `max_points` covers the task estimate.
3. The runtime is installed and its declared capabilities pass probing.
4. The grant matches reality: `ro` is enforceable and `rw` has write tools.
5. The runtime can honor model selection if a model is specified.
6. The role's skills can be delivered at their required fidelity.

Never make one provider the framework default. Prefer a different runtime or
model family for adversarial review when genuine independence matters.

## Run one delegated task

```bash
dacli context <ref>
dacli spawn --task <ref> --role <role> --worktree --claim <narrow-path> \
  --pr --detach
dacli wait
dacli sync
dacli task show <ref>
dacli ship --project <project> --verify "<command>" --into main --pr --dry-run
dacli ship --project <project> --tasks <ref> --verify "<command>" \
  --into main --pr --push
```

Use `--worktree` for parallel work and a truthful, narrow `--claim` for every
write task. A worktree separates current files and branches; a claim prevents
agents from producing overlapping future merges. Use `dacli commit` inside an
agent worktree so identity, role, and task trailers are recorded.

Use `agents` and `logs -f` to observe. Use `wait` to finalize detached runs.
Run `sync` after read-only agents because their writes are proposals in the
event log until an owner materializes them.

For direct work performed by the current agent, use `task claim`. Do not claim a
task as the parent immediately before `spawn`; spawn records the child claim.
The dry run currently cannot preview an explicit active `--tasks` window even
though the real pipeline accepts that wave before integrating it; issue #651
tracks that disagreement. Preview the project wave, inspect pending proposals,
then use the explicit task window on the real command.

## Run a wave or loop

Preview scheduling and cost before spawning:

```bash
dacli critical-path --project <project>
dacli next --project <project> --parallel <width>
dacli loop --project <project> --width <width> --max-cycles <n> --dry-run
dacli loop --project <project> --width <width> --max-cycles <n>
```

For unattended work, add `--yolo` deliberately and retain a hard cycle,
rolling-token, and `--halt-after-idle` bound. Stop safely at the next checkpoint
with:

```bash
touch .dacli/STOP
```

Do not interrupt during an unknown landing state unless safety requires it.
Inspect `loop status`, agent runs, PR state, and trunk progress before resuming.

## Verify claims, not activity

Run the repository's documented verification bar. For dacli itself:

```bash
gofmt -l .
go vet ./...
golangci-lint run
go test ./...
```

When adding a regression test, mutate or revert the protected behavior and
confirm the test fails for the intended reason. Record anything unverified.
Never check an acceptance box merely because an agent reported success.

Use independent verification when the consequence warrants it:

```bash
dacli verify --task <ref> --panel <runtime-a>,<runtime-b> --require 2
dacli accept <ref> --require-independent --require-verify \
  --verify "<command>"
```

## Collaborate through GitHub without losing the record

Use GitHub issues, PRs, reviews, checks, and Projects as the human-visible
surface. Keep dacli's task/event record synchronized so scheduling, attribution,
and verification remain truthful.

Preview external mutations first:

```bash
dacli github doctor
dacli github pull <project> --dry-run
dacli github push <project> --dry-run
dacli github sync <project> --dry-run
dacli integrate --tasks <refs> --into main --pr --no-merge
```

Use `dacli pr`, `integrate`, and `ship` rather than raw merge commands because
they carry task acceptance, findings, verdicts, CI state, and landing events.
Push requested branches and expose the PR URL; do not publish releases or tags
without explicit authority.

## Respect the command contract

| Exit | Meaning | Response |
|---:|---|---|
| 0 | Operation completed | Continue |
| 1 | Operational failure | Diagnose the reported cause |
| 2 | Usage error | Correct arguments |
| 3 | Policy refusal | Stop; follow the named remedy; never retry unchanged |
| 4 | Object not found | Re-check workspace, project, branch, and reference |

Capture the command's own exit status when testing it; a pipeline reports the
last process's status unless the shell is configured otherwise.

## Finish with an evidence-backed handoff

Before declaring completion:

1. Re-read the task and check only criteria actually satisfied.
2. Run targeted tests and the repository verification bar.
3. Run `dacli doctor` after unusual state changes.
4. Confirm the intended commit is on the intended branch and contains only
   claimed paths.
5. Push the branch and create or update the GitHub PR when requested.
6. Wait for required CI; use dacli's landing path; confirm trunk actually moved.
7. Close or synchronize the GitHub issue only after the deliverable landed.
8. Report changes, verification, PR/issue state, and anything still unverified.

The product of dacli is a record that agrees with reality. Prefer an honest
failure or empty cycle over a successful-looking report that cannot be proved.
