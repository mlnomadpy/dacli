# Teams: roles, escalation, and spawning

**Status:** the pure engine (`internal/team`: scope globs, escalation routing, WIP limits) is implemented and tested; commands are stubs; spawning depends on [RUNTIMES.md](RUNTIMES.md), which is spec only.

How a tree of agents is organized, what each one is allowed to touch, and what happens when one hits something outside its competence.

---

## 1. The rule that keeps roles from being cosplay

**A role must change what an agent can do, not just what it calls itself.**

Prepending "You are a senior frontend engineer with 10 years of experience" to a prompt is theater. It costs tokens, it flatters the model, and it changes nothing mechanical. Role-play prompting produces agents that *describe* their work in the register of the role while behaving identically.

A role in `dacli` is a set of four concrete affordances:

| Affordance | Effect |
|---|---|
| **Skills** | Which skill files load into the agent's context at spawn |
| **Scope** | Which paths it owns, as globs — and which it is fenced out of |
| **Shortcuts** | Which commands are in its toolkit |
| **Escalation** | Who it must ask when the work falls outside scope |

If a proposed role changes none of these, it should not exist. `dacli lint` flags a role whose only distinguishing feature is its name.

Roles are also the SPM answer to task assignment: a WBS assigns **roles, not individuals**. The role outlives any particular agent, which is exactly right when agents are ephemeral and get spawned by the dozen.

## 2. Role definition

`.dacli/roles/<name>.md`:

```markdown
---
id: role-backend
kind: role
name: backend
summary: server-side Go, the data layer, and the CLI surface
skills: [pallas-kernels, jax-ecosystem]
scope:
  - internal/**
  - cmd/**
out_of_scope:
  - internal/legacy/**
shortcuts: [test, build, lint, bench]
grant: rw
escalate_to: [architect, human]
wip: 3
runtime: claude-code
fallback: [codex]
max_turns: 6
budget: 40000
---

Owns everything behind the CLI boundary. Does not touch generated
protobuf or the legacy adapter, both of which are frozen.
```

**`out_of_scope` beats `scope`.** A deny that a broader allow can override is not a boundary.

**Empty `scope` means no fence**, deliberately. Most projects want one role and no walls, and forcing everyone to enumerate scope up front produces wrong globs written to satisfy a linter.

**`wip`** caps concurrent agents in the role. This is Kanban's work-in-progress limit, and it is the only thing standing between an enthusiastic parent agent and thirty children contending over four files. It converts the **Burning Across** anti-pattern from something `dacli doctor` detects afterward into something the spawner refuses up front.

### Rosters are per-project

There is no universal role set. A web product wants `frontend` / `backend` / `sre` / `reviewer`. A research repo wants something else entirely — for a paper repo, roles like `theorist` (skills: `math-kernel-theory`, `math-paper-audit`; scope: `papers/**/main.tex`), `experimentalist` (skills: `jax-ecosystem`, `kaggle-cli-experiments`; scope: `experiments/**`), and `figure-editor` (skills: `tikz-figures`; scope: `plots/**`) carve the work along the lines that actually exist.

`dacli init --roster software|research|solo` seeds a starting set. Editing it is expected.

## 3. Escalation: why there is no chat room

You asked for a Slack — somewhere agents can talk things through the way an engineering team does. I want to push back on that specifically, because I think it's the one part of this design that would actively hurt.

**Four reasons a chat channel between agents fails:**

1. **Agents are agreeable.** What makes human design discussion valuable is somebody willing to hold a position under social pressure. Models mostly aren't; two agents "discussing" converge fast and confidently, and the convergence carries no more information than either started with. You get the *appearance* of review with none of the adversarial pressure that makes review work.
2. **A conversation has no definition of done.** Every other object in `dacli` has a completion criterion — acceptance boxes, a cursor, an applied flag. A chat has none, so it runs until something arbitrary stops it. Unbounded token spend with no stopping rule is exactly what a budget-aware tool should refuse to build.
3. **Chat is ephemeral; the whole thesis here is durability.** A decision reached in a channel that never becomes a decision note may as well not have happened. And if you're writing the note anyway, the chat was pure overhead.
4. **It reproduces the thing Slack is criticized for**, in a system that has no social reason to need it. Human teams use chat partly because the alternative is interrupting someone. Agents have no feelings to spare.

**What engineering teams actually get from Slack** is two separable things: *asynchronous unblocking* and *ambient awareness*. Both decompose into primitives `dacli` already has substrate for, without the failure modes:

- **Unblocking → a typed help request.** Addressed to a role, carrying a specific question, requiring an answer that lands as a durable note.
- **Ambient awareness → views over the event log.** `dacli standup`, `dacli status`, `dacli threads`. The channel ergonomics with none of the conversation.

So: help requests, not chat.

### The help request

```
dacli ask --about t-004 --need internal/ledger/batch.go \
  "Does the nightly batch job write balances directly, bypassing the service layer?"
```

This appends a `help` event and blocks the asking task. `dacli team route` resolves who should answer by walking the escalation chain from the asker's role until it finds a role whose scope covers `--need`.

An answer:

```
dacli answer <id> --as decision "Yes — it writes directly. The shim must wrap
the batch path too. Rejected: wrapping only the service layer."
```

The answer becomes a `decision` or `finding` note attached to the task, so it enters every future context brief in scope. **The question is transient; the answer is permanent.** That asymmetry is the entire design.

`dacli threads` renders help-request chains as a channel-like view for humans reading along. It is a projection over the event log, not a separate store.

### Two-tier escalation

The chain terminates at `human`, and that is a normal outcome rather than a failure:

1. **In-tree** — ask a role that covers the path. This is asking a teammate.
2. **Out-of-tree** — nothing in the roster covers it, so `Route` returns `ErrNoOwner` and the request escalates to a human. This is filing a ticket.

For the second tier, a GitHub issue is the right vehicle and I'd wire it as an optional adapter (`dacli escalate --github`): it reaches a human where they already are, it survives the session, and it has its own notification path. But it must stay optional — requiring a GitHub remote for a local workspace would break the no-infrastructure story, and `dacli init` deliberately doesn't need one.

**An agent tree that can never say "nobody here owns this" will instead have somebody guess, and the guess ships.** `ErrNoOwner` is a feature.

One routing subtlety the error message must handle: escalation follows *declared* `escalate_to` edges, so a role can own the path while being unreachable from the asker's chain — an org chart with a missing edge. That is a configuration gap, not a dead end, and the failure must say so: *"backend owns `internal/ledger/` but is not in frontend's escalation chain — add it to `escalate_to`, or route through architect."* A bare "no owner" would send someone hunting for a role that already exists.

## 4. Spawning

```
dacli spawn --role backend --task t-004 [--budget 30000]
```

One call does what currently takes four:

1. Checks the role's WIP limit and refuses if exceeded.
2. Mints a child identity with the role's default grant, attenuated against the parent's (a read-only parent cannot spawn a read-write child, whatever the role says).
3. Assembles the context brief for the task.
4. **Compiles the role's skills for the target runtime** ([SKILLS.md](SKILLS.md)) — native skill dir, context-file section, or brief-inline, whichever the adapter supports, with any degradation announced — and prepends the shortcut catalog.
5. Prints the child's token and brief.

Step 4 is what makes `skills:` honest across runtimes: the field used to assume every CLI speaks one vendor's skill system. Skills are authored once in the workspace and delivered in whatever form the child's runtime can actually load.

Attenuation still wins over role configuration. A role's `grant: rw` is a *ceiling request*, not an override — otherwise the capability system would be bypassable by writing a role file.

**`runtime:`, `model:`, and `max_points:` are the cost-policy fields**, and they are implemented: `spawn`/`supervise` resolve the runtime from the role when no `--runtime` is passed, route `model:` onto the runtime's declared `model_flag` (a runtime without one makes routing *inoperative and announced*, never silently ignored), and enforce `max_points:` as a **seniority gate** — a junior role capped at 3 points is mechanically refused the Te-8.7 migration (exit 3, naming a heavier role), and refused unestimated work outright: a capped role takes only work whose size somebody stated. The economics this encodes: reviewers get the expensive model because judgment is where model quality pays; juniors get the cheap model because the seniority gate guarantees they only ever see work the cheap model can carry.

```
dacli role add junior   --grant rw --runtime cc --model haiku --max-points 3
dacli role add reviewer --grant ro --runtime cc --model opus --wip 1
dacli spawn --task 014 --role junior          # runtime, model, and size cap all from the role
dacli spawn --task 014 --role reviewer --review --pr-number 12
```

**Git and PR discipline ride the prompt registry** ([PROMPTS.md](PROMPTS.md)): every rw child receives `git_workflow` — branch per task (`dacli/NNN-slug`), commit-per-logical-change with the task ref, red-suite-means-the-box-stays-unchecked, and either the full push-plus-`gh pr create` flow (`--pr`, with the PR URL reported as a finding — an unrecorded PR does not exist) or an explicit do-not-push. `--review` children receive `review_workflow` instead: judge the `gh pr diff` against the brief's acceptance criteria rather than taste, file every defect twice (dacli finding and PR comment), and approve only what they would stake their verdict on. A reviewer's sandbox must allow `Bash(gh:*)` alongside the dacli binary, or the child is instructed to report the refusal and stop.

**`kind:` places a role in the project lifecycle** — researcher, planner, designer, implementer, reviewer — and is what phase gating acts on. A template's stages declare a `phase:` and an `allow:` list of role-kinds ([TEMPLATES.md](TEMPLATES.md)); `dacli spawn` refuses a role whose kind the current phase disallows, so **you cannot spawn an implementer while the project is still in discovery** — advance the gate first. A role with no `kind` opts out (works in any phase); solo/untemplated projects are never gated. The current phase and its appropriate work appear in every brief, so agents know when to research versus build without being told per-task.

```
dacli project add "New product" --slug np --template product   # starts in discovery
dacli role add scout --kind researcher --grant ro
dacli role add builder --kind implementer --grant rw
dacli spawn --task 001 --role builder   # refused: discovery allows researcher, reviewer
dacli spawn --task 001 --role scout     # allowed
# ...advance the gates...
dacli spawn --task 001 --role builder   # allowed once the phase reaches implementation
```

**`runtime:` selects which coding-agent CLI the child actually runs on** ([RUNTIMES.md](RUNTIMES.md)). Two consequences worth designing around:

- For a spawned child, the runtime's own sandbox enforces the grant, so `dacli`'s cooperative permission model becomes genuinely enforced for exactly these agents. A runtime that cannot enforce read-only causes a refusal to spawn, not a downgrade.
- An `implementer` role and a `reviewer` role should default to **different** runtimes. Review by the same model that wrote the code is the author grading its own homework with the same blind spots; different vendors fail in uncorrelated ways, which is the only cheap source of real independence.

`fallback` is opt-in per role, deliberately. A silent vendor switch would let a verification panel quietly collapse onto one runtime while still looking diverse.

### Team views

| Command | Shows |
|---|---|
| `dacli team` | Roster: roles, active agents per role, WIP headroom |
| `dacli team route <path>` | Who owns this path, and the escalation chain to reach them |
| `dacli agent tree` | Lineage, grants, and write attribution |
| `dacli standup` | Per-agent: done, next, impediments — derived from the event log |
| `dacli threads` | Open help requests and their answers |
| `dacli doctor` | Anti-patterns, including role-specific ones |

### Two anti-patterns that roles create

Specialization has costs, and `dacli doctor` should look for both:

- **Silos** — an agent whose `context` calls never surfaced a sibling's findings. Role fences make this more likely, not less, which is the honest trade for scoping.
- **Groupthink**, in its one real agent form — a panel of verifiers that agree because they were handed identical prompts and identical context. The counter is prompt and context diversity, not a facilitation technique, so `doctor` flags verification events that shared a brief rather than suggesting a meeting.
