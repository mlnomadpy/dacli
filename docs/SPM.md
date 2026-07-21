# The SPM layer

**Status:** the pure engines (`internal/spm`: ambiguity scanner, PERT + Cone, CPM with typed dependencies) are implemented and tested; every command that consumes them is a stub.

`dacli` is not a neutral container for whatever an agent decides to write down. Its object model encodes software product management frameworks directly, so an agent that uses `dacli` at all is organizing its work the SPM way without being told to.

This document is the mapping. It is opinionated on purpose: the failure mode it targets is an agent that writes a vague task, spawns three children on non-critical work, discovers the blocking unknown at 80% budget, and ships something nobody asked for. Every framework below is here because it prevents a specific one of those.

---

## The honest part: not everything ports

These frameworks were designed for human teams, and human teams have bottlenecks agents don't have — limited working memory, morale, communication cost, politics. Porting all of it uncritically would be cargo-culting. Three tiers:

### Tier 1 — Ports directly

The framework works on an agent tree for the same reason it works on a human team, with no reinterpretation.

WBS · Critical Path Method · task dependency types · MoSCoW · INVEST · acceptance criteria · risk impact×likelihood · risk-value ordering · GQM · FURPS+ · review severity classification · ambiguity categories · glossary · traceability IDs · vision-vs-scope · Cone of Uncertainty · PERT.

These are all about *structure of the work*, which is substrate-independent.

### Tier 2 — Ports with reinterpretation

The shape is right, the unit is wrong. The substitution in each case is **tokens for time**:

| Human framework | Agent reading |
|---|---|
| Sprint / iteration | One bounded agent work session |
| Velocity (points per sprint) | Tasks completed per 100k output tokens |
| Time box (2–4 weeks) | Token box (budget ceiling per session) |
| Iteration burndown (hours remaining vs. days) | Points remaining vs. tokens spent |
| Daily scrum (3 questions) | Per-agent status roll-up from the event log |
| Sprint retrospective | Findings harvest when a task subtree completes |
| Sustainable pace | Budget headroom for the verification pass |

Note that Tier 2 is nearly free: the event log already timestamps and attributes every write, so burndown, velocity, and standup are *derived*, not separately bookkept. An agent never fills in a status report.

Miller's Law (5–9 items in working memory) reinterprets too, and usefully: it is the reason the context assembler caps the constraint section rather than emitting every decision ever made. An agent handed 40 constraints will silently drop most of them, exactly like a human would.

### Tier 3 — Does not port. Deliberately absent.

Kerth's artifacts contest, the emotions seismograph, appreciations circles, secret-ballot safety polls, "creating safety," most of the retrospective readying course, the individual anti-patterns (micromanagement, seagull management, intellectual violence), groupthink counters that rely on social dynamics, and the whole apparatus around developer morale.

These address human psychological and political failure modes. An agent has none of them. Implementing them would be ceremony that costs tokens and buys nothing. `dacli` does not have a `dacli retro --appreciations`.

One partial exception worth naming: **Groupthink has a real agent analogue** — a panel of verifiers that all agree because they were given identical prompts and identical context. The counter is the same in spirit (independent, diverse framing) but the mechanism is prompt diversity, not a facilitation technique. `dacli` surfaces it as a `doctor` check on verification events sharing a brief, not as a meeting.

---

## Framework → object mapping

### Vision vs. Scope (Wiegers) → Project

`project.md` gains `## Vision` (the why, long-term) alongside `## Goal`, plus `## Out of scope` as an explicit list.

The out-of-scope list is the scope-creep defense, and for agents it does real work: it is emitted into **every** context brief, so a child agent that starts drifting toward an adjacent problem has already been told, in its own context, that the problem is out of bounds. Scope creep in agent trees is not a client asking for more — it is a child agent helpfully deciding to also refactor the thing next door.

### MoSCoW → task priority

`priority: must | should | could | wont` in task frontmatter. `dacli task list` sorts by it, and `dacli next` will never recommend a `could` while a `must` is open.

This directly targets **Cart Before the Horse**, which is the single most common agent planning failure: given a decomposed problem, agents reliably start with the tractable, interesting piece rather than the load-bearing one.

`wont` tasks are kept, not deleted. A recorded out-of-scope decision stops the next agent from re-proposing it.

### INVEST → `dacli lint`

Each letter becomes a mechanical check on a task:

| Letter | Check |
|---|---|
| **I**ndependent | No `blocked_by` cycle; warn at depth > 3 in the parent chain |
| **N**egotiable | Body states a goal, not a diff. Heuristic: flags tasks whose body is mostly code |
| **V**aluable | Has a `so that` clause or an `about` link to a project goal |
| **E**stimatable | Has an estimate; `xl` is flagged as an epic to decompose |
| **S**mall | Estimate within one session's budget |
| **T**estable | Has at least one `## Acceptance` checkbox |

A task failing **T** is the highest-value lint in the tool. A subagent given a task with no acceptance criteria cannot know when to stop, and will either stop too early or keep going until it runs out of budget.

### The 11 ambiguity categories → `dacli lint --ambiguity`

Implemented for real, in `internal/spm/ambiguity.go`. Runs the categories over task titles, bodies, and acceptance criteria.

This is the highest-leverage framework in the entire skill for agent work, and it is nearly free to implement — word lists and a few patterns. Vague task text is the dominant cause of subagent failure. "Handle the errors properly" fails three categories at once (vague verb *handle*, vague noun *errors*, qualifier *properly*) and will produce three different implementations from three different agents.

Human teams tolerate ambiguity because a developer walks over and asks. A subagent does not walk over and ask. It guesses, confidently, and the guess is the deliverable.

**Scope policy, because the noise profile is real:** several moderate-severity categories (qualifiers like *all*/*only*, positional *after*/*before*, temporal *until*/*when*) are routine in ordinary prose — "move the file after tests pass" flags *after* while being perfectly clear. A linter that fires on every sentence gets ignored, and an ignored linter is worse than none. So the default scope is asymmetric: **titles and acceptance criteria lint at moderate-and-above; task bodies lint at major only.** The places where ambiguity becomes a wrong deliverable get the strict pass; the places where prose is just prose don't train agents to skip the output. `--strict` widens everything.

### PERT + Cone of Uncertainty → estimates as ranges

Estimates are three-point, never scalar:

```yaml
estimate: {optimistic: 2, probable: 5, pessimistic: 14}
```

`Te = (To + 4Tm + Tp)/6` and `σ = (Tp − To)/6` are computed, and the project's `stage:` (`definition` | `elicitation` | `approach` | `design`) applies the Cone of Uncertainty multiplier on top. A 6-unit estimate at elicitation stage reports as 3–12, not 6.

Why this matters for agents specifically: an agent asked for an estimate produces a confident scalar with no error bars, every time. Forcing three points forces the pessimistic case to be *stated*, which is where the unexamined risk lives. The number that actually predicts overrun is `Tp`, and nothing else in an agent's workflow ever asks for it.

### WBS + CPM → the parallelism scheduler

This is the framework with the largest payoff, and its agent reading is different from its human one.

For a human team, the critical path tells you where a delay hurts. For a parent agent, the critical path tells you **which tasks to spawn subagents on first**. Tasks with slack can wait; tasks with zero slack are the ones gating completion. Fanning out onto slack tasks while the critical path sits idle is wasted concurrency — and it is what an agent does by default, because it fans out on whatever decomposed most cleanly.

`dacli critical-path <project>` returns the zero-slack chain and per-task slack. `dacli next --parallel N` returns the N tasks worth spawning on right now.

Dependency types are recorded (`FS`, `SS`, `FF`, `SF`) because `SS` is what makes two tasks genuinely parallel-safe, and that distinction is invisible in a plain `blocked_by`.

Degradation, per the announce-every-omission axiom: CPM needs durations, so when tasks lack estimates `dacli next` falls back to MoSCoW-then-dependency order and says so — a silent fallback would let a priority-sorted list masquerade as a critical path.

### Risk-value matrix → ordering

High-risk/high-value first. For agents this is the *fail-fast budget rule*: the task most likely to invalidate the plan should run before the budget is committed, not after. An agent that saves the risky integration for last discovers at 80% spend that the whole decomposition was wrong.

New object: `projects/<slug>/risks/<slug>.md` with `impact`, `likelihood`, computed `rank` (1/2/3), `indicators`, and `action`. Rank 1 and 2 require an action plan; `dacli lint` flags a rank-1 risk with no action. Rank 3 is monitored, not planned.

Risks are emitted into context briefs — a child working near a known risk is told about it and told what the indicator looks like.

### Review severity → findings

`finding` notes carry `severity: major | moderate | minor`, using the review-technique definitions (major = fix not obvious, needs exploration; moderate = fix clear but needs review; minor = obvious or unnecessary).

This maps exactly onto what a code-review agent produces, and it lets the context assembler rank findings by severity rather than only recency — which is the crude ranking flagged as an open question in DESIGN.md § 11.2. Severity is a partial answer that costs nothing.

### GQM + FURPS+ → metric notes

`note_kind: metric` with structural `## Goal` / `## Question` / `## Metric`. FURPS+ is the question-generation prompt.

The rule this enforces is Basili's: you cannot choose a metric before stating the goal. An agent asked to "add some metrics" will produce whatever is easy to count — the LOC failure mode, in a new costume. Requiring the goal and question first makes the bad metric visibly unjustified.

### Defect analysis → derived from events

Findings-opened vs. findings-resolved over token spend. The crossover (the **software barrier**) is computable from the event log with no extra bookkeeping, and for an agent tree it signals the same thing it does for a team: stop finding, start fixing.

### Anti-patterns → `dacli doctor`

Detectors over the event log. Every one of these is a real, observed agent failure:

| Anti-pattern | Detector |
|---|---|
| Analysis Paralysis | Findings accumulating, zero tasks moved to `done` |
| Cart Before the Horse | A `could`/`should` task active while a `must` task sits `open` |
| Gold Plating | Events on a task far exceeding its estimate, with all acceptance boxes already checked |
| Death March | Task `active` across N sessions with no acceptance progress |
| Fire Drills / Heroics | Nearly all events attributed to one agent while siblings are idle |
| Silos | An agent whose `context` calls never included sibling findings |
| Burning Up | Total remaining points rising — scope being added mid-session |
| Burning Across | Points flat — tasks started, none finished |
| Over-Engineering | Task body grows while acceptance criteria don't |

`dacli doctor` is the piece with no equivalent in any competing tool, because no competing tool has an attributed event log to run it over.

### Glossary → `projects/<slug>/glossary.md`

A shared term list, emitted into every context brief. Cheap, and it directly attacks the vague-noun ambiguity category by giving the project's terms one definition that every agent in the tree sees.

### Traceability → `traces:`

Tasks and requirements carry `traces: [path/to/file.go, path/to/test.go]`, satisfying the traceability quality criterion and letting `dacli` answer "which requirement does this code serve" — the question that makes a change reviewable.

---

## New commands

| Command | Framework |
|---|---|
| `dacli lint` | INVEST, requirements quality criteria, decision completeness |
| `dacli lint --ambiguity` | The 11 ambiguous-language categories |
| `dacli estimate <task>` | PERT 3-point + Cone of Uncertainty range |
| `dacli critical-path <project>` | CPM: zero-slack chain and per-task slack |
| `dacli next [--parallel N]` | Risk-value ordering + MoSCoW + critical path |
| `dacli risk add\|list` | Impact×likelihood matrix |
| `dacli doctor` | Anti-pattern detection over the event log |
| `dacli burndown` | Points remaining vs. tokens spent |
| `dacli velocity` | Tasks per 100k tokens, trailing sessions |
| `dacli standup` | Per-agent roll-up: done / next / impediments |
| `dacli retro <task>` | Findings harvest: went well / didn't / improve |
| `dacli wbs <project>` | Work breakdown tree |
| `dacli glossary` | Project term list |

---

## What an agent actually does with this

The intended loop, which is the argument for the whole design:

1. `dacli context <task>` — get the brief: goal, scope boundary, constraints, decisions already made, sibling findings, live risks, glossary.
2. `dacli lint <task>` — if the task is ambiguous or untestable, **fix that before working**, not after.
3. `dacli critical-path` — if fanning out, spawn on zero-slack tasks first.
4. Work. Append findings as they happen, with severity.
5. `dacli doctor` — before declaring done, check whether the work drifted into a known anti-pattern.
6. `dacli retro` — harvest findings into durable notes so the next agent inherits them.

Step 2 is the one that pays for the tool. Everything upstream of the work is cheaper than everything downstream of getting it wrong.
