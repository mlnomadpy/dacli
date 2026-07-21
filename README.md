# dacli

**Context management for hierarchies of coding agents.** Markdown on disk, folders for structure, a CLI and an MCP server as the two front ends.

An agent that spawns subagents has one hard problem: each child starts blind. It re-reads the codebase, re-derives decisions its siblings already made, and re-attempts work that already failed. `dacli` is the shared workspace that fixes this — a durable, human-readable project state that any agent in the tree can query, and that the parent can slice down to exactly the context a given child needs.

Everything is markdown with YAML frontmatter and `[[wikilinks]]`. That means git diffs it, `grep` searches it, GitHub renders it, Obsidian opens the workspace as a vault with no plugin, and you can fix it by hand when an agent writes something stupid.

## The core idea

```bash
dacli context task/042 --budget 4000
```

One command returns a single self-contained markdown brief: the task, the goal it serves, the constraints that bound it, the decisions already made that it must not relitigate, and the findings its siblings have reported — trimmed to fit a token budget, most-relevant-first.

You hand that to a subagent instead of the whole repo. That is the product. Task tracking is the substrate that makes it possible.

## Status

**Pre-alpha — design + skeleton.** The on-disk format and command surface are specified ([DESIGN.md](DESIGN.md), [docs/FORMAT.md](docs/FORMAT.md)); command bodies are stubs. The format spec is the stable part; treat the Go API as unstable.

The docs index — every document, one line each, with an honest status label — is [docs/README.md](docs/README.md). Start with [DESIGN.md](DESIGN.md) for the why, [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for the normative shape (axioms, layers, build order, the canonical brief), and [docs/WALKTHROUGH.md](docs/WALKTHROUGH.md) to watch one task travel the whole system end to end.

## Three surfaces, one store

- **Obsidian** is where humans read and write documents.
- **GitHub** is where humans coordinate and work becomes visible outside the session.
- **`dacli`** (CLI and MCP) is where agents work.

One markdown store underneath. None of them owns it. GitHub is a *projection* that can be deleted and regenerated — local markdown stays the source of truth, because `dacli context` is the hot path and must never touch the network.

## Install

```bash
go install github.com/mlnomadpy/dacli/cmd/dacli@latest
```

## Quickstart

```bash
# In your project root
dacli init --name "payments-refactor"

dacli project add "Migrate billing to the new ledger" --slug ledger
dacli task add "Audit every write path into balances" --project ledger
dacli note add decision "Ledger writes stay synchronous" --project ledger \
  --body "Async was rejected: reconciliation cost exceeds the latency win."

# Parent agent mints a read-only child identity
TOKEN=$(dacli agent spawn --role auditor --grant ro)

# Child agent, in its own process
DACLI_AGENT=$TOKEN dacli context task/001 --budget 3000
DACLI_AGENT=$TOKEN dacli status
```

## Command surface

| Command | Purpose |
|---|---|
| `dacli init` | Create a `.dacli/` workspace |
| `dacli context <ref>` | **Assemble a scoped brief for an agent** |
| `dacli status` | Tree-wide project state, one screen |
| `dacli agent spawn` | Mint a child agent identity + capability |
| `dacli agent tree` | Show the agent lineage and who wrote what |
| `dacli project add\|list\|show` | Projects |
| `dacli task add\|list\|show\|claim\|done\|block` | Tasks |
| `dacli note add` | Decisions, findings, references |
| `dacli queue add\|next\|advance` | Ordered step lists |
| `dacli events tail` | Append-only write log |
| `dacli sync` | Owner applies pending child events |
| `dacli mcp serve` | Same core, exposed as MCP tools |

Plus the SPM layer (`lint`, `estimate`, `critical-path`, `next`, `wbs`, `risk`, `doctor`, …) and the team layer (`spawn`, `team`, `role`, `ask`, `run`, …). Run `dacli help` for the full surface.

## Teams and shortcuts

**Roles** organize the tree — and the rule that keeps them from being cosplay is that *a role must change what an agent can do, not just what it calls itself*. A role determines which skills load at spawn, which paths are in scope, which shortcuts are reachable, and who to escalate to. It also carries a Kanban WIP limit, so `spawn` refuses the thirty-children-over-four-files situation up front instead of `doctor` diagnosing it after.

**Escalation is a typed help request, not a chat channel** — a deliberate reversal of the obvious design, argued in [docs/TEAM.md § 3](docs/TEAM.md). Agents are agreeable, so two of them "discussing" converge without adding information; a conversation has no completion criterion, which is disqualifying in a budget-aware tool; and chat is ephemeral in a system whose whole thesis is durability. The question is transient, the answer becomes a decision note and enters every future brief. The chain terminates at `human` — a tree that can't say "nobody here owns this" will have somebody guess instead.

## Runtimes — spec only, not built

`dacli` spawns its agents by invoking coding-agent CLIs (Claude Code, Codex, Gemini CLI, opencode, …) and supervising them against a task's acceptance criteria. Design in [docs/RUNTIMES.md](docs/RUNTIMES.md). The decisions that matter:

- **Results come back through the workspace, not stdout.** Children report by calling `dacli`, so results are format-independent, uniformly attributed, and *survive the child being killed mid-run* — which is exactly when partial work is most valuable. Parsing each vendor's output format would make schema-chasing the permanent central problem.
- **Adapters are declarative files, probed rather than trusted.** `dacli runtime doctor` verifies each adapter's assumed flags against the installed binary. Shipped flag sets are starting points, not facts.
- **Spawning makes permissions genuinely enforced** for spawned children, since `dacli` sets the runtime's own sandbox flags. A runtime that can't enforce read-only causes a refusal, never a silent downgrade.
- **Heterogeneity is the feature.** A verification panel drawn from one model is a single point of failure wearing several hats; different vendors fail in uncorrelated ways.
- **Skills are authored once and compiled per runtime** ([docs/SKILLS.md](docs/SKILLS.md)) — native skill dir where one exists, a managed context-file section where one doesn't, brief-inline as the floor, every degradation announced. A skill's scripts compile to effect-gated shortcuts on targets that can't carry executables.

**Shortcuts** are named command templates ([docs/SHORTCUTS.md](docs/SHORTCUTS.md)). The token saving is real but minor; the point is that a shortcut is a *memoized derivation* — the flags and the working directory somebody already paid to discover, made durable instead of evaporating with the session. Every parameter is POSIX-quoted (values carry model-generated text; concatenation is an injection vector), and `read`/`write`/`destructive` effects gate execution so `deploy` is never one token away from `test` in a list the model is skimming.

## The SPM layer

`dacli` is not a neutral container for whatever an agent writes down. The object model encodes software product management frameworks directly, so an agent using `dacli` at all organizes its work the SPM way without being told to: MoSCoW priority, INVEST checks, PERT three-point estimates, the Cone of Uncertainty, WBS, CPM with typed dependencies, the impact×likelihood risk matrix, GQM metrics, review severity, and the eleven categories of ambiguous language.

Full mapping in [docs/SPM.md](docs/SPM.md) — including an explicit list of the frameworks that **do not** port to agent work and are deliberately absent. Three things it argues:

- **Ambiguity linting is the highest-leverage check here.** Human teams tolerate vague requirements because a developer walks over and asks. A subagent doesn't walk over and ask — it guesses, confidently, and the guess is the deliverable. `dacli lint --ambiguity` flags "handle all the errors properly" on three categories at once.
- **The critical path is a parallelism scheduler.** For a human team, CPM says where a delay hurts. For a parent agent it says which tasks to spawn children on *first* — fanning out onto slack tasks while the critical path idles is wasted concurrency, and it's the default agent behavior.
- **Tokens replace time.** Velocity, burndown, and time boxes port only with that substitution, and they come nearly free: the event log already timestamps and attributes every write.

## Design decisions worth knowing before you adopt it

- **Permissions are cooperative, not enforced.** A subagent with shell access can edit the markdown directly and bypass `dacli` entirely. The capability system prevents well-behaved agents from clobbering each other; it is not a security boundary. See [DESIGN.md § Permission model](DESIGN.md#permission-model) for what an enforced version would require.
- **No shared file is ever edited by two agents.** Cross-agent writes are append-only events, one file per event, ULID-named. Concurrency is handled by never contending, not by locking.
- **dacli does not execute anything.** Queues are ordered markdown step lists. The agent runs the steps; `dacli` tracks position. A task tracker and a job runner are different products.
- **There is no Obsidian plugin.** The workspace conforms to Obsidian's conventions, so `File → Open vault` on your project root works today.

## License

MIT
