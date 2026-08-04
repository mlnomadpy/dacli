---
name: dacli
description: Use dacli to run a repository with a team of AI agents — plan work as tasks, size it, route it to the cheapest capable model, run agents in parallel worktrees, and land it through PRs with real gates. Triggers when the user asks to build something substantial, run agents in parallel, manage a backlog, or when dacli is installed and the work is bigger than one sitting.
---

# dacli

`dacli` turns a repository into a workspace a **team of agents** can work in:
tasks with acceptance criteria, isolated worktrees, PRs, quality gates, and a
governed loop. You are the operator. Your job is to set direction and let the
machinery do the coordination.

**Reach for it when** work is bigger than one sitting, splits into independent
pieces, or needs a record of *why* things were decided. **Don't** for a
one-file fix — `dacli` is coordination overhead, and overhead you don't need is
just cost.

Everything is markdown under `.dacli/`. Nothing is hidden; read the files when
unsure.

> **The workspace is files in the tree, so it forks with the branch.** A task
> filed on a branch does not exist on trunk: `spawn --task 250` answers `not
> found: 250` while the record sits in an unmerged PR. `next`,
> `critical-path` and `burndown` each compute over one checkout, so every branch
> reports a different project state and none of them is wrong. **File tasks on
> trunk.** Seq numbers are allocated by scanning the tree too, so two branches
> will hand out the same number to different tasks.

## Exit codes are a contract

| Code | Meaning | Action |
|---|---|---|
| 0 | worked | continue |
| 2 | usage error | fix the command line |
| **3** | **refused by policy** | **an ANSWER — never retry it** |
| 4 | not found | check the ref |
| 1 | other failure | read the message |

**Retrying a 3 is the most expensive mistake you can make here.** It will never
succeed. Read the refusal — it names what to do instead.

## Start

```bash
dacli new "Recipe API" --goal "REST API for storing recipes" --stack python
```
Greenfield: creates the workspace, a project with a filled goal/spec/
architecture, a detected stack, a **CI workflow**, and a dependency-ordered
starter backlog. Add `--gitignore-workspace` to keep the `.dacli/` workspace out
of trunk so the generated repo is code, not bookkeeping — its full history then
lives on the record branch (`dacli ship --record-branch <branch>`).

```bash
dacli adopt --provision-roles
```
Existing repo: maps the codebase, seeds tasks from TODOs, provisions a roster.

## The working loop

```bash
dacli next                      # what to work on now (MoSCoW, then critical path)
dacli context <ref>             # the brief you would hand an agent
dacli spawn --task <ref> --role fixer --worktree --pr --detach
dacli agents --tail             # live: who is running, and their last line
dacli wait                      # block until the wave finishes
dacli accept <ref> --verify "go test ./..."
```

`--worktree` is not optional for parallel work: it gives each agent its own
directory **and branch**, which is the whole reason two agents can run at once
without clobbering each other. `--detach` returns immediately so you can spawn
a wave.

## Landing work — through dacli, not through `gh`

```bash
dacli integrate --tasks 205,243 --into main --pr    # merges only checks-passing PRs
dacli integrate --tasks 205 --into main --pr --auto # hand a pending PR to GitHub auto-merge
dacli integrate --tasks 205 --into main --pr --no-merge  # open it, stop for review
dacli ship --into main --pr                          # the whole wave: accept → integrate → record
```

`dacli integrate` is the merge path because it does the bookkeeping raw
`gh pr merge` skips: it resolves each task's branch, posts recorded verdicts to
the PR, gates on `gh pr checks`, and records the merge as an event against the
task. **A merge that leaves no event did not happen** as far as the workspace is
concerned.

Three things that will bite you:

- **The branch name is a lookup key**, not a label. `integrate` finds
  `dacli/<seq>-<slug>` using the task's *own* slug, and **silently skips**
  anything else — you get "0 PRs opened" with no error. Let `spawn --worktree`
  name it, or copy the slug exactly.
- **`integrate` and `ship` refuse from any branch but `--into`.** Check out
  trunk first.
- **Merge overlapping PRs one at a time.** Two PRs touching one file merge
  cleanly in sequence and conflict as a batch. Merge, confirm the next is still
  mergeable, then take it.

**There is a role for this.** `integrator` (`role_kind: reviewer`) exists to take
green PRs to trunk without you babysitting: `dacli spawn --task <ref> --role
integrator --detach`, from the trunk checkout, **without `--worktree`** — a
worktree is a different branch, and `integrate` would refuse. If you find
yourself merging by hand, you skipped a role that already does it.

## Running a sprint

```bash
dacli loop --project core --width 4 --max-cycles 1
```
One cycle = build (N agents in parallel) → wait → land → review → retro. The
review phase files the next evidence-backed task, so the backlog regenerates.

**Always bound it.** `--max-cycles N`, or keep `--no-progress-halt` above zero.
A perpetual loop with no stop condition is refused unless you pass `--yolo`.
`touch .dacli/STOP` halts it; that stop **latches**, so an agent cannot undo it.

## Make parallelism real

```bash
dacli task estimate <ref> --estimate 2,4,8    # optimistic,probable,pessimistic
dacli critical-path                            # schedule + slack; ★ = critical
dacli next --parallel 6                        # what can run concurrently
```

**Size the backlog or you lose the schedule.** Without estimates,
`critical-path` refuses and `next` silently falls back to MoSCoW-then-sequence —
the one ordering that cannot tell you what runs concurrently. Three points, not
one: the third point is where the risk is stated.

## Spend cheap models on easy work

```bash
dacli team assign <ref>     # → the cheapest role whose capacity covers this task
```

Roles declare a `model` tier and a `max_points` capacity. `team assign` picks
the lowest-cost role that fits the task's Te and the phase's allowed kind. A
task above every cap means **decompose it**, not "use a bigger model and hope".

**Treat the recommendation as a floor, not an answer.** The ranking knows cost
and capacity; it does not know blast radius. A one-line gitignore edit and a
subtle idempotency bug against a live API both come back `--role junior` if both
fit the cap. Override when the *consequence* of getting it wrong is large.

> **Check the role can actually do the work before you spawn it.** A role
> declares a `grant` *and* a `runtime`, and nothing cross-checks them. `junior`
> shipped with `grant: rw` and `runtime: cc` — and `cc`'s allowlist is
> Read/Grep/Glob/LS plus the dacli binary, no Edit, no Write. Spawning an
> implementation task on it burns a whole run discovering the agent cannot write
> a file. `grep runtime .dacli/roles/<name>.md`, then check that runtime's
> allowlist in `.dacli/runtimes/`.

Cost controls: `--max-tokens N` per spawn (refuses above budget),
`--window-tokens N --budget-window 24h` for a rolling ceiling, `dacli calibrate`
for measured cost per band, `--advise` to preview cost.

> `--advise` means the same thing everywhere: it **previews and returns** — on
> `loop`, `spawn`, and `supervise` alike it prints the calibrated sizing (and, on
> spawn/supervise, the task's taint status) and launches nothing. Re-run without
> `--advise` to actually spawn. (It used to spawn on `spawn` — dacli 232 made the
> flag honest.)

## Quality gates that check software, not paperwork

```bash
dacli stage <project>          # current stage and every gate check
dacli stage advance <project>  # advances only if the gate opens
```

Attach a template (`--template tdd`) and gates become executable:
`command: go test ./...`, `coverage: 80 <cmd>`, `artifact: go.mod`. The `tdd`
template gates on build, suite, and a coverage floor — note that `go test`
passes trivially with **zero** tests, which is exactly why the coverage gate
exists.

## Recording decisions is the highest-value thing you do

```bash
dacli note add decision "chose X" --project p --rejected "Y" --because "<why>"
dacli note add finding "<what>" --project p --origin file.go:42
```
Decisions enter every future brief, so the next agent does not re-propose what
you already rejected. That is the single biggest waste dacli removes.

## Reading state

```bash
dacli status        # tree-wide, one screen
dacli overview      # human-first summary
dacli doctor        # anti-patterns AND data-integrity problems
dacli dashboard     # local web UI: projects, burndown, live swarm, roster
dacli blame <file>  # which agent and role wrote each line
dacli agent show <id>  # role, lineage, runs, events
```

Run `dacli doctor` after anything unusual — it catches orphaned tasks, duplicate
files, and task records that lost their frontmatter.

## Best practices

1. **Acceptance criteria decide everything.** They are what an agent is measured
   against and what `accept --verify` proves. Vague criteria produce vague work.
   `dacli lint` flags ambiguous language before you spend tokens on it.
2. **Let the review phase find work.** Don't hand-write a 40-task backlog from
   imagination; seed a few, run a cycle, and let evidence file the rest.
3. **Prefer `--verify` on every close.** `accept --require-verify` makes an
   unverified close impossible — use it when the record matters.
4. **Independence beats volume.** `accept --require-independent` stops the agent
   that did the work from also certifying it.
5. **Read a brief before a big spawn** (`dacli context <ref>`). If it doesn't
   contain what the agent needs, the agent won't have it either.

## Anti-patterns

- **Retrying a refusal (exit 3).** It is an answer.
- **Unbounded loops.** Always `--max-cycles` or a progress halt.
- **Spawning without `--worktree` for parallel work** — the agents will collide,
  and a branch may get checked out in your main tree. (The one exception is the
  `integrator`, which must be on trunk.)
- **`git add -A`, or `dacli commit` without `--no-add`, while a wave is
  running.** The default stages everything, and everything includes the other
  agents' in-flight edits — which then land under your commit message and your
  attribution. Stage the paths you touched.
- **Merging by hand when `integrator` exists.** Same class of mistake as writing
  code yourself instead of spawning a fixer.
- **Closing tasks without verification** — the record then claims something
  nobody checked.
- **Ignoring `doctor`.** Silent workspace corruption is the failure mode that
  hides longest.
- **Letting worktrees accumulate.** `spawn --worktree` creates one per task and
  nothing reclaims them; `git worktree list | wc -l` and `du -sh
  .dacli/worktrees` are worth a glance after a long run.
- **Using dacli for a trivial change.** The overhead is real; skip it.
