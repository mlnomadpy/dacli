# dacli — Design

Status: **draft**. This document is the contract. The Go code is an implementation of it and yields to it when they disagree.

Layering, build order, interface contracts (exit codes, JSON, MCP), and the canonical brief example live in [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md), which is normative and wins over this document where they overlap. The 2026-07-21 design audit is [docs/REVIEW.md](docs/REVIEW.md).

---

## 1. Problem

A parent agent decomposes work and spawns children. Each child starts with an empty context window and gets, at best, a paragraph of prompt. The observed failure modes:

1. **Rediscovery.** Every child re-reads the same files to learn the same things.
2. **Relitigation.** A decision the parent made in turn 3 is invisible to a child spawned in turn 40, which proposes the rejected alternative.
3. **Sibling blindness.** Child 3 spends its whole budget on an approach child 1 already proved doesn't work.
4. **Result evaporation.** A child's findings live only in its final report, which the parent summarizes lossily and then loses.
5. **Budget waste.** The safe move is to over-brief every child, so every child pays for context it doesn't use.

These are all the same problem: **there is no durable, queryable, sliceable project state shared across the agent tree.**

`dacli` is that state. The primary operation is not "track a task" — it is "given this agent and this task, produce the smallest brief sufficient to do the work."

## 2. Non-goals

- **Not a workflow engine.** `dacli` supervises **agent processes it spawned**, one per task, each bounded by a budget, a turn cap, and a timeout. It owns no DAG of jobs, schedules nothing against a clock, and never decides that arbitrary work should run. Queues stay ordered step lists with a cursor, executed by the agent.

  The defensible line, and the one to hold: **`dacli` runs agents, not work.** The moment it grows a cron trigger, a job dependency graph, or a step that is neither an agent nor a named shortcut, it has become a CI system with a markdown skin and should stop.

  *This non-goal originally read "not a job runner: no process execution, retries, or timeouts." Shortcuts (§ 11) narrowed it once; runtimes (§ 12) narrowed it again. Two erosions is where you restate the boundary instead of patching it, so the wording above replaces the original rather than qualifying it. The history is kept because the first version was load-bearing in the queue design.*
- **Not a security boundary.** See § 6.
- **Not a replacement for git.** The workspace is committed to the repo. History is git's job.
- **Not an issue tracker for humans.** Humans can read and edit it — that is a design constraint — but the ergonomics target agents.

## 3. On-disk layout

The workspace is a `.dacli/` directory at the project root. Full file-level spec in [docs/FORMAT.md](docs/FORMAT.md).

```
.dacli/
  config.yml                      # workspace id, name, format version
  agents/
    root.md                       # the agent that ran `dacli init`
    a-01J8F3K9.md                 # each spawned agent, with parent link
  projects/
    ledger/
      project.md                  # goal, constraints, success criteria
      tasks/
        open/     001-audit-write-paths.md
        active/   002-add-ledger-shim.md
        blocked/
        done/     000-inventory-callers.md
      notes/
        decisions/  d-sync-writes.md
        findings/   f-legacy-batch-job.md
        refs/       r-ledger-rfc.md
  queues/
    release-checks.md
  events/
    2026/07/21/01J8F3KA-a01J8F3K9-claim.md
```

### Why folders for status

Status is folder position (`tasks/open/`, `tasks/active/`, …), not a frontmatter field. Three reasons: `ls tasks/active` is a status query with zero tooling; Obsidian's file tree becomes a kanban board; and a `git diff` of a status change is a rename, which reads correctly in review. The cost is that a status change is a `git mv` — acceptable, and only the task owner performs it (§ 6).

### Why markdown

Objectively worse data modeling than SQLite, and the right call anyway. Agents read and write it natively with no serialization step. It diffs. It greps. It renders on GitHub. When an agent corrupts it, a human repairs it in an editor instead of a REPL. The query patterns here are shallow and the datasets are small — hundreds of tasks, not millions — so the performance argument for a database never binds.

## 4. Object model

Five core types carry the collaboration; later layers added satellites (risk, role, shortcut, and — spec only — runtime and template) that follow the same frontmatter rules. All carry `id`, `created`, `created_by`, and arbitrary `tags`.

| Type | Lives in | Purpose |
|---|---|---|
| **Project** | `projects/<slug>/project.md` | A goal with constraints and success criteria. The root of a context tree. |
| **Task** | `projects/<slug>/tasks/<status>/NNN-<slug>.md` | A unit of work. May have a `parent` task. Has exactly one `owner` agent. |
| **Note** | `projects/<slug>/notes/<kind>/` | `decision`, `finding`, or `ref`. The durable output of agent work. |
| **Queue** | `queues/<slug>.md` | Ordered steps + a cursor. |
| **Agent** | `agents/<id>.md` | Identity, parent, granted capability, lineage. |

Relations are `[[wikilinks]]` in frontmatter and body. A link that resolves to nothing is not an error — it is a marker for something worth writing later. This matches Obsidian's semantics exactly and is deliberate.

**Decisions are first-class and this is the point.** A `decision` note records what was chosen, what was rejected, and why. It is the single highest-value item in a child's context brief, because it is the thing children most reliably get wrong when they don't have it.

## 5. Context assembly — the core algorithm

`dacli context <ref> [--budget N] [--depth D] [--format md|json]`

Produces one self-contained markdown document. Sections are emitted in **fixed priority order**, and trimming under a budget removes from the *bottom*, so the highest-value content is never the part that gets cut:

1. **Task** — full frontmatter and body. Never trimmed. If this alone exceeds the budget, error rather than truncate.
2. **Goal chain** — the parent project's vision, goal, and success criteria, plus ancestor task titles from root to this task. Establishes *why*.
3. **Scope boundary** — the project's `## Out of scope` list. Cheap, and it is the only scope-creep intervention that lands *before* the tokens are spent.
4. **Constraints** — project constraints plus every `decision` note reachable from the task or its ancestors. Prevents relitigation.
5. **Risks** — open rank-1 and rank-2 risks with their indicators, so a child knows what the early warning looks like.
6. **Glossary** — the project term list. Counters the vague-noun ambiguity category at the source.
7. **Sibling findings** — `finding` notes on sibling and recently-completed tasks, ranked by severity then recency. Prevents duplicated dead ends.
8. **Linked refs** — `[[wikilink]]` targets at depth ≤ D (default 1), excerpted.
9. **Recent events** — the last N events touching this task subtree.

Sections 3–6 are small and near-constant in size, which is why they sit above findings: a boundary, a constraint set, and a term list are worth more per token than one more anecdote.

**Miller's Law applies to the trimmer.** An agent handed 40 constraints silently drops most of them, exactly as a human would, so the constraint and risk sections cap at a single-digit count and report the overflow rather than emitting everything and hoping. A brief is a working-memory budget, not an archive.

Budget accounting uses a character-count heuristic (`chars/4`) by default; `--tokenizer` may later select a real one. Every trim is announced inline (`<!-- dacli: 3 findings omitted, budget -->`) rather than silently applied — a brief that looks complete but isn't is worse than one that admits its gaps.

`--format json` returns the same sections structured, for the MCP path.

## 6. Permission model

**This is cooperative, not enforced. Stating that plainly is part of the design.**

A subagent has shell access — that is what makes it useful — so it can open the markdown in an editor and bypass `dacli` entirely. Any claim that `dacli` *prevents* a subagent from writing would be false.

What the capability system actually buys you: well-behaved agents that go through `dacli` cannot accidentally clobber each other, and every write is attributed and auditable.

### Mechanism

- `dacli init` creates the **root agent** with capability `rw`.
- `dacli agent spawn --grant ro` mints a child agent, records `parent:` in `agents/<id>.md`, and prints an opaque token.
- A child passes its token via the `DACLI_AGENT` environment variable.
- Children may spawn their own children, but **cannot grant a capability exceeding their own** — monotonic attenuation. An `ro` agent's descendants are all `ro`.
- Every write records the acting agent id. `dacli agent tree` shows lineage and attribution.

### What `ro` agents can do

Read everything, and **append events** (§ 7). An `ro` agent is not mute: it reports findings, claims tasks, and proposes status changes. It just cannot mutate an object another agent owns. This is the important nuance — a read-only agent that cannot report results is useless.

### What an enforced version would require

A long-lived daemon owning the only writable copy of the workspace, with children given a read-only bind mount (or a separate uid with POSIX permissions) and the daemon as the sole write path over a unix socket. That is a substantially larger project and should not be attempted until the cooperative model has proven the ergonomics are right.

### Spawning makes enforcement partly real

There is a cheaper path to real enforcement, and it arrives with runtimes ([docs/RUNTIMES.md](docs/RUNTIMES.md) § 8): when `dacli` launches the child process itself, it controls that runtime's sandbox flags. A read-only agent gets the runtime's own read-only mode, which the child cannot escape by declining to call `dacli`.

Three limits, all of which have to be stated wherever this is claimed:

- It applies **only to agents `dacli` spawned**. An agent that runs `dacli` from a shell someone else started is exactly as cooperative as before.
- It is **only as strong as the vendor's sandbox**, and those vary.
- A runtime that cannot enforce read-only causes a **refusal to spawn**, not a downgrade. A capability that silently isn't there is worse than one never claimed.

## 7. Concurrency — append-only events

Two agents editing one markdown file will corrupt it. Rather than lock, `dacli` arranges for contention never to arise.

**Rule: each object is written only by its owner. Everything else is an event.**

An event is a new file, never an edit: `events/YYYY/MM/DD/<ULID>-<agent>-<kind>.md`. ULIDs are lexicographically sortable by creation time, so the directory listing *is* the ordered log. Two agents writing simultaneously create two different files. There is no shared mutable state, therefore no race, therefore no lock.

Event kinds: `claim`, `release`, `finding`, `propose-status`, `comment`, `block`.

`dacli status` and `dacli context` read task files **and** fold in pending events, so a child's finding is visible tree-wide the instant it is written — no sync step needed for *reads*.

`dacli sync`, run by an object's owner, materializes pending events into the object itself (moving a task folder on an accepted `propose-status`, appending a finding to the task body) and marks them applied. Events are never deleted; `dacli events compact --before <date>` archives them.

The one genuinely shared mutable file is nothing — there isn't one. `config.yml` is written once at init.

## 8. Two front ends, one core

`internal/` holds all logic and has no knowledge of either front end.

- **CLI** (`cmd/dacli`) — for humans and for agents whose only affordance is Bash. Stable text output; `--format json` everywhere for parsing.
- **MCP server** (`dacli mcp serve`) — the same operations as typed tools with schemas. Agents that speak MCP should use this: no stdout parsing, no quoting bugs, and the tool descriptions themselves teach the agent the workflow. Specified in [docs/MCP.md](docs/MCP.md): a tiered surface (fourteen core tools plus a `cli` escape hatch), identity bound at launch so tokens never enter transcripts, and policy refusals returned as results — not errors — so nothing retries a "no".

Designing for both from the start is cheap; retrofitting an MCP server onto a CLI whose logic lives in command handlers is not.

## 9. Obsidian

No plugin. The workspace conforms to Obsidian's conventions — YAML frontmatter, `[[wikilinks]]`, folder hierarchy — so opening the project root as a vault works immediately. Graph view renders the decision/finding link structure for free. Any integration code here would be maintenance burden for near-zero marginal value.

## 10. The SPM layer

`dacli` is not a neutral container for whatever an agent decides to write down. Its object model encodes software product management frameworks directly — MoSCoW priority, INVEST checks, PERT three-point estimates, the Cone of Uncertainty, WBS, CPM with typed dependencies, the impact×likelihood risk matrix, GQM metrics, review severity, and the eleven ambiguity categories — so an agent that uses `dacli` at all is organizing its work the SPM way without being told to.

Full mapping in [docs/SPM.md](docs/SPM.md), including an explicit account of which frameworks **do not** port to agent work and are deliberately absent. Three claims from it are load-bearing here:

1. **Ambiguity linting is the highest-leverage check in the tool.** Human teams tolerate vague requirements because a developer walks over and asks. A subagent does not walk over and ask — it guesses, confidently, and the guess is the deliverable.
2. **The critical path is a parallelism scheduler, not a status report.** For a human team, CPM says where a delay hurts. For a parent agent it says which tasks to spawn children on *first*: fanning out onto slack tasks while the critical path idles is wasted concurrency, and it is the default agent behavior.
3. **Tokens replace time.** Velocity, burndown, and time boxes port only with that substitution — and they come nearly free, because the event log already timestamps and attributes every write. Nothing is separately bookkept.

`internal/spm` holds the framework computations and is pure: no disk, no workspace, no agent identity. Ambiguity scanning, PERT, the Cone, and CPM (all four dependency types, with slack and cycle detection) are implemented and tested there.

## 11. Teams and shortcuts

Two subsystems layered on the same substrate. Full designs in [docs/TEAM.md](docs/TEAM.md) and [docs/SHORTCUTS.md](docs/SHORTCUTS.md).

**Roles** (`.dacli/roles/<name>.md`) organize the agent tree. The rule that keeps them from becoming cosplay: *a role must change what an agent can do, not just what it calls itself.* A role determines which skills load at spawn, which paths are in scope, which shortcuts are reachable, and who to escalate to. Prompt-level role-play — "you are a senior frontend engineer" — changes nothing mechanical and is not what this is. Roles also carry a Kanban WIP limit, which turns Burning Across from something `doctor` detects afterward into something `spawn` refuses up front.

Role grants are a *ceiling request*, not an override: attenuation still wins, or the capability system would be bypassable by writing a role file.

**Escalation is a typed help request, not a chat channel.** This was a deliberate reversal of the obvious design, and the reasoning is in docs/TEAM.md § 3. Briefly: agents are agreeable, so two of them "discussing" converge without adding information; a conversation has no completion criterion, which is disqualifying in a budget-aware tool; and chat is ephemeral in a system whose entire thesis is durability. What teams actually get from Slack decomposes into asynchronous unblocking (a help request) and ambient awareness (views over the event log) — both available without the failure modes. The question is transient; the answer is promoted to a decision note and enters every future brief in scope.

The escalation chain terminates at `human`, and `ErrNoOwner` is a feature: a tree that can never say "nobody here owns this" will instead have somebody guess, and the guess ships.

**Shortcuts** (`.dacli/shortcuts/<name>.md`) are named command templates. The token saving is real but minor; the actual value is that a shortcut is a *memoized derivation* — the flags, the working directory, the environment variable somebody already paid to discover, made durable and reviewable instead of evaporating with the session. They also make the tree's command surface auditable, which nothing else does.

Two safety properties are non-negotiable and implemented in `internal/shortcut`: every parameter is POSIX-quoted unless a committed file declares `raw`, because parameter values carry model-generated text and concatenation is an injection vector; and effects (`read`/`write`/`destructive`) gate execution against the caller's grant, with destructive requiring explicit confirmation so that `deploy` is never one token away from `test` in a list the model is skimming.

Shortcuts are advertised in briefs by use count and truncated with the omission announced — an unused shortcut is a permanent per-brief tax, so the catalog gets the same budget discipline as everything else. Most shortcuts should arrive by *promotion* from repeated commands in the event log rather than by an agent predicting what it will repeat, which it cannot do.

## 12. Runtimes

**Specification only; nothing implemented.** Full design in [docs/RUNTIMES.md](docs/RUNTIMES.md).

`dacli` spawns its agents by invoking coding-agent CLIs — Claude Code, Codex, Gemini CLI, opencode, others — and supervises them against a task's acceptance criteria. This inverts the original assumption that an agent already existed and chose to call `dacli`.

Five load-bearing decisions:

1. **The return channel is the workspace, not stdout.** Parsing each vendor's output format would make schema-chasing the central engineering problem. Instead children report by calling `dacli`, exactly as they do now — so results are format-independent, uniformly attributed, visible while streaming, and *survive a child being killed mid-run*, which is precisely when partial work is most valuable. stdout is a debugging transcript.
2. **Adapters are declarative files, probed rather than trusted.** `.dacli/runtimes/<name>.md` declares binary, flags, and capabilities; `dacli runtime doctor` verifies them against the installed binary and caches per-machine results. The landscape moves too fast for compiled-in adapters, and shipped flag sets are assumptions until probed.
3. **Degradation is explicit.** A runtime without session resume makes turn *n* cost *n*× the context; a runtime without usage reporting makes every cost figure an estimate and labels it as one. Observability may degrade quietly; safety properties never do.
4. **Enforcement gets real** for spawned agents (§ 6).
5. **Heterogeneity is the feature.** A verification panel drawn from one model is a single point of failure wearing several hats. Different vendors fail in uncorrelated ways, which is the only cheap source of genuine independence — and it is the real counter to the one agent-shaped form of Groupthink.

The supervision loop terminates because each turn is evaluated against acceptance criteria written *before* the work started, with turns and budget both capped. That external criterion is exactly what distinguishes it from the agent chat this design rejects in § 11.

**Tasks are assumed small, well-scoped, and single-turn by default**, and that assumption does real work: it removes the need for session resume in the common case, bounds cost-estimation error to one task, and is what makes acceptance criteria checkable enough for the loop to terminate. Multi-turn supervision is the exception, and reaching turn 3 is a signal the task was mis-sized. The floor is the brief itself — goal chain, constraints, and glossary run 1–2k tokens per task, so slicing finer than a few multiples of that pays the brief tax for work that is mostly overhead.

Slash commands (`/review`, `/commit`, repo-defined ones) are invoked rather than reimplemented — the shortcut argument one level up, reusing prompt engineering a vendor maintains and tests. Tasks name abstract verbs; adapters map them to literal commands; unmapped verbs fall back to a plain prompt, announced.

The remaining hard problem is cross-tree prompt injection: a hostile file read by one child becomes context for every sibling, and small tasks make the per-child blast radius smaller while making the propagation surface wider. Stated as open in RUNTIMES.md § 18 rather than smoothed over.

## 13. Projects, templates, and GitHub

**Specification only.** See [docs/TEMPLATES.md](docs/TEMPLATES.md) and [docs/GITHUB.md](docs/GITHUB.md).

**Templates** ([TEMPLATES.md](docs/TEMPLATES.md)) define what kind of project this is: required documents, staged gates, a role roster, and a definition of done. The gate predicate vocabulary is small and non-scriptable on purpose — a gate that can run arbitrary code stops being auditable. The predicate that carries the weight is *filled*, not *present*: a section still containing `TBD`, or failing the ambiguity linter at major severity, does not satisfy a gate. "Make the thing better and handle the edge cases properly" is exactly as empty as "TODO", and this is where the SPM layer pays for itself.

Gates are also bureaucracy, so `solo` (one stage, no gates) is the default, heavier templates state their cost, and `dacli doctor` flags a project spending more agent turns on gate documents than on tasks — Viewgraph Engineering, made mechanically detectable. Shipping stage gates without that detector would be selling the disease with the cure.

**GitHub** ([GITHUB.md](docs/GITHUB.md)) is a *projection*, never a backend. Local markdown stays the source of truth because `dacli context` is the hot path and must never touch the network, because contention-freedom depends on local ULID-named files rather than a shared mutable store, because rate limits would cap the fan-out this tool exists to enable, and because git already versions everything.

The elegant part is inbound: a human commenting on an issue, closing it, or moving a card becomes an **event**, structurally identical to a child agent appending a finding — an outside party contributing to an object it does not own. Inbound sync therefore needs no new concurrency machinery, and a human closing an issue *proposes* a status change that the owner applies, preserving the invariant that only an owner rewrites an object.

Two safety points: mirroring to a public repo is a disclosure event, so it requires explicit per-project confirmation; and enabling inbound sync on a public repo lets strangers write text into your agents' context, which must be a decision rather than a side effect.

Obsidian remains conform-don't-integrate (§ 9), with two zero-cost additions: generated index notes, and optional Dataview-compatible inline fields so vault users get live task boards for free.

The three surfaces, stated cleanly: **Obsidian** is where humans read and write documents, **GitHub** is where humans coordinate, **`dacli`** is where agents work — over one markdown store that none of them owns.

## 14. Prior art and differentiation

Task Master AI, Backlog.md, Beads, and Claude Code's built-in todo state all cover "markdown-ish task tracking for agents." That is not the differentiator and pitching it that way loses.

The differentiator is two things neither of which anything above has:

1. The **agent hierarchy** — attenuating capabilities, lineage-attributed writes, contention-free concurrent access from a whole tree of agents, and budget-aware context slicing per child.
2. The **SPM layer** (§ 10) — an opinionated object model that makes the well-managed structure the path of least resistance, plus `dacli doctor`, which detects management anti-patterns over the event log. No competing tool can do that, because none has an attributed event log to run detectors over.

## 15. Direction: from organizing to compounding

**Proposed, not accepted** — argued in [docs/PROPOSALS.md](docs/PROPOSALS.md). The v1 goal candidate: every run leaves the workspace measurably smarter than it found it. Four learning loops, all reading data the log already collects — lessons distilled across projects, estimates calibrated against actuals, any past run replayable as what-did-the-agent-know, and taint traced from any suspect source to every brief it reached. One decision *is* urgent regardless of acceptance: the four capture fields (recorded briefs, stamped actuals, event `origin:`, note `scope:`) must land in v0.1, because history not captured is history lost.

## 16. Open questions

1. **Token counting.** The `chars/4` heuristic is wrong per-model. Worth a real tokenizer, or is announced-trim honesty enough?
2. **Finding relevance.** § 5 step 4 currently means "same project, recent." Sibling findings are the highest-signal section and the crudest ranking. Embeddings would help and would drag in a dependency that undermines the zero-infrastructure story.
3. **Cross-repo workspaces.** Multi-repo work implies a workspace outside any one repo. Deferred; `.dacli/` is repo-local for v1.
4. **Event volume.** A long-lived tree of chatty agents produces a lot of small files. Compaction policy is sketched, not designed.
5. ~~**Task id collisions.**~~ **Resolved** (2026-07-21 review): the true id is `t-<ULID>` assigned at creation, collision-free under concurrency; `NNN` is a display alias in the filename, assigned by the project owner at sync — a single allocator with no race. References accept ULID, `NNN`, or slug. See docs/FORMAT.md § Task.
