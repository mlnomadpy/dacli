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
starter backlog.

**The workspace stays out of trunk by default.** `new`, `init` and `adopt` all
gitignore `.dacli/` and record `record_branch: dacli-record` in the workspace
config; `ship` commits the full trajectory there. The two are one decision —
ignoring the workspace *without* a record branch would delete its history, not
tidy it. Read it back with `git log dacli-record`. Opt out with
`--gitignore-workspace=false`.

This matters because records carried on a branch fork with it: a task closed on
a task branch is invisible on trunk until its record merges, so the loop
re-picks finished work.

```bash
dacli adopt --provision-roles
```
Existing repo: maps the codebase, seeds tasks from TODOs, provisions a roster.

> **Migrating a repo that tracked `.dacli`:** the untracking commit removes it
> from the index, so git deletes it from your working tree on the next pull.
> Nothing is lost — restore with `git archive origin/dacli-record .dacli | tar -x -C .`
> (not `git checkout`, which re-stages the files and undoes the untracking).

## The working loop

```bash
dacli next                      # what to work on now (MoSCoW, then critical path)
dacli context <ref>             # the brief you would hand an agent
dacli spawn --task <ref> --role fixer --worktree --pr --detach
dacli agents --tail             # live: who is running, their state and last line
dacli catchup --since 20m       # what your siblings filed since your brief
dacli wait                      # block until the wave finishes
dacli accept <ref> --verify "go test ./..."
```

`--worktree` is not optional for parallel work: it gives each agent its own
directory **and branch**, which is the whole reason two agents can run at once
without clobbering each other. `--detach` returns immediately so you can spawn
a wave.

## Running a wave — let dacli do the coordination

This is the section people skip, then spend the afternoon resolving merge
conflicts by hand. **Every part of organising a wave is a command.** If you are
doing it in your head, you are doing it worse and leaving no record.

```bash
dacli next --parallel 6                 # 1. WHAT runs concurrently — CPM, with slack per task
dacli team assign <ref>                 # 2. WHO — cheapest capable role, and why
dacli spawn --task <ref> --role <r> --worktree --detach \
            --claim "internal/brief"    # 3. WHAT IT MAY TOUCH — the conflict gate
dacli agents                            # 4. who is live
dacli wait                              # 5. block, and FINALIZE each run's outcome
dacli accept <ref> --verify "<cmd>"     # 6. close with evidence
dacli integrate --tasks <refs> --into main --pr   # 7. land
```

**`--claim` is the whole answer to "how do I stop agents colliding".** It
reserves paths for a live agent, and an overlapping spawn is refused before it
starts:

```
path-claim conflict: live agent a-maintainer-me4vk0 already claims
"internal/brief" and you claim "internal/brief" — narrow your scope,
or `dacli wait 01KZ785H1A` first
```

It names the holder, the path, and both recoveries. Claim the package an agent
will edit, not the whole repo — an over-broad claim serialises a wave that could
have run wide, and no claim at all means dacli cannot protect anything.

Two rules that follow:

- **Group by claim, not by intuition.** Deciding "these six touch disjoint
  packages" yourself is the arbitration `--claim` performs — except yours is
  unrecorded and unenforced. Hand it the claims and let it refuse.
- **A refused claim is scheduling information.** It means that task belongs in
  the *next* wave. Queue it; do not narrow the claim to something dishonest just
  to get it running.

`dacli next --parallel N` picks the set for you once tasks are sized — it is
critical-path ordered and prints the slack, so you know which task delays the
whole wave if it slips. Picking the wave from your own reading of the backlog
instead is how the critical path gets ignored.

### After the wave

Use **`dacli wait`**, not a poll of `dacli agents`. `wait` is what *finalizes*
each run — it writes the outcome, and a child that left no events and no checked
acceptance is recorded as **`no visible result`**. Poll `agents` instead and
that classification never runs: a silent agent looks exactly like a working one,
and you find out hours later. `agents` is for watching; `wait` is for closing.

**A read-only agent cannot write the workspace — it proposes.** Auditors,
reviewers and any `grant: ro` role file findings and status changes as *events*.
They are not in the record until you run **`dacli sync`**, which applies them.
If an audit "produced nothing", run `sync` before believing it.

**WIP limits count agents, not live agents.** A role with `wip: 1` whose slot is
held by a long-finished agent is permanently unspawnable. `dacli agent retire
<id>` frees it; `dacli doctor` is where you notice.

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

**`accept` moves the task file from `open/` to `done/`, and that move is a
working-tree change somebody has to commit.** Run `accept` and `integrate` by
hand, forget the commit, and `doctor` reports the task as existing in two status
folders — a small mess that recurs every wave. `ship` closes that gap: accept →
integrate → **commit the `.dacli` record** → push, and the third step is the one
you will otherwise keep dropping.

Reach for `ship` on a young project. On one with a long history, run
`ship --dry-run` first and read the integrate line: **`ship` hands `integrate`
the entire done set, not the wave it just closed.** At 250 done tasks that is a
250-ref command whose branches are mostly gone (harmlessly skipped) but which
can also re-merge an old branch that still exists locally. When the dry-run
looks like that, close the wave by hand — `accept --verify`, `integrate --tasks
<the refs you actually landed>`, then commit `.dacli` yourself.

Four things that will bite you:

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

## Running sprints unattended — `dacli loop`

Everything in the wave section above is one cycle. `dacli loop` runs that cycle
back to back, with no human between them:

```bash
dacli loop --project core --width 4 --max-cycles 8      # 8 sprints, then stop
dacli loop --project core --width 4 --yolo \
           --window-tokens 3000000 --budget-window 24h  # until the backlog drains
```

One cycle = **review → plan → implement → test → land → retro**. The review
phase files the next evidence-backed task, so the backlog regenerates and the
loop feeds itself. This is the mode to use when you want the backlog drained
rather than a wave approved: hand it the width and the bound, and leave.

**`--yolo` is what removes the between-cycle pause.** Without it the loop stops
and waits for you at each checkpoint, which is the right default when you are
watching and the wrong one when you have said "keep going until it's done".

**Bound it anyway.** A perpetual loop with *no* stop condition is refused
outright. Give it at least one of:

| Bound | What it does |
|---|---|
| `--max-cycles N` | hard stop after N sprints |
| `--window-tokens N --budget-window 24h` | sleeps when the rolling spend is exhausted |
| `--no-progress-halt N` (default 3) | halts after N cycles that land nothing — the thrash guard |

The governor sits in front of every cycle and it **idles on an empty backlog
rather than inventing work**, so "until done" terminates instead of degenerating
into busywork.

**Stopping it: `touch .dacli/STOP`.** The stop *latches* — an agent cannot undo
it, and the loop halts at the next checkpoint rather than mid-merge. That is the
kill switch to reach for, not `Ctrl-C` into an unknown state.

`dacli loop --dry-run` prints the whole plan — phases, width, the tasks it would
build — without spawning anything. Read it once before a long unattended run;
it is also where you notice that the wave it picked is not the wave you meant.

`dacli loop status` shows the running or last loop's cycle count, trunk marker,
tokens spent this window, and ready-backlog size.

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

## Closing a task means the work actually landed

```bash
dacli accept <ref> --verify "go test ./..." --require-verify
```
`--verify` proves the TREE is healthy; it does not prove THIS task's work is in
it. So accept also checks whether the task's branch reached trunk, and warns
when it did not — under `--require-verify` that becomes a refusal. A build
passing while the deliverable never merged is how a run once reported 15 of 21
done with the commands missing entirely.

`--allow-unlanded` closes one deliberately. Either way the outcome is written
to the task's log, so the record never implies a landing nobody confirmed.

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

## Preview anything that writes somewhere you cannot un-write

```bash
dacli github push <project> <refs...> --dry-run
dacli loop --dry-run
dacli ship --dry-run
dacli worktree prune --dry-run
```

A `--dry-run` on a GitHub command is not politeness, it is the only chance you
get: an issue published to a public repo can be closed but never unpublished.
The preview prints counts per object kind, and that is where you catch a scope
you did not intend — a **windowed** push still mirrors decisions and findings
project-wide, so `--tasks 268` can mean *one task issue and 157 decision
issues*. You only see that in the dry-run.

Same shape for `worktree prune`: the dry-run classifies each candidate as
merged-into-trunk or run-finished, so you can read the list before deleting 97
checkouts.

## When a wave does not go cleanly

```bash
dacli loop status               # rollup + the next step for each bad outcome
dacli pr status --task <ref>    # merging, behind base, or really conflicted?
dacli threads                   # questions agents are blocked on
```
The rollup names a recovery per outcome rather than only counting them. And
`pr status` distinguishes a branch that is merely **behind** its base (merge
trunk in; nothing to resolve) from one that genuinely **conflicts** — GitHub
reports both as unmergeable, and triaging one as the other wastes the time
this command exists to save.

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
- **Spawning a wave without `--claim`.** A worktree stops agents overwriting each
  other's files *now*; a claim stops them writing the same file at all, which is
  what turns into a merge conflict later. Grouping the wave by package in your
  head is the same arbitration, done worse and recorded nowhere.
- **Polling `agents` instead of calling `wait`.** `wait` finalizes each run;
  polling never does, so a child that produced nothing is indistinguishable from
  one still working until you happen to open a transcript.
- **Believing a read-only agent that "produced nothing"** before running
  `dacli sync`. Its findings are sitting in the event log, unapplied.
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
- **Letting worktrees accumulate.** `spawn --worktree` creates one per task, and
  left alone they pile up — this repo reached 103 checkouts and 2.4 GB before
  anyone looked. `dacli worktree prune --dry-run` classifies each candidate as
  merged-into-trunk or run-finished so you can check the list before deleting
  anything; the loop runs the real prune each cycle.
- **Using dacli for a trivial change.** The overhead is real; skip it.
