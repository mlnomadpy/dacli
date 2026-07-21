# Runtimes: driving coding-agent CLIs

**Status: specification. Nothing here is implemented.**

`dacli` spawns its agents by invoking coding-agent CLIs — Claude Code, Codex, Gemini CLI, opencode, and others — supervises them against a task's acceptance criteria, and collects their work through the workspace.

---

## 1. What this changes

Every previous version of `dacli` assumed the agent already existed and chose to call `dacli`. The tool was a passive store: agents brought their own context problem and `dacli` answered it.

This inverts that. `dacli` now *creates* the agent process. That makes it an orchestrator, and three things follow immediately:

1. **`dacli` owns the child's lifecycle** — budget, timeout, termination, retry on transient failure. It has to, because nothing else will.
2. **Permission enforcement can become real** (§ 8). This is the biggest win and it retires the most uncomfortable caveat in DESIGN.md § 6.
3. **The heterogeneity is the point, not an obstacle** (§ 10). Different vendors' models fail differently, which is the only cheap source of genuine independence in a verification panel.

It also means `dacli` acquires the failure modes of a process supervisor. § 11 and § 17 are about paying that honestly rather than pretending it didn't happen.

### The small-task assumption

This spec is written assuming **tasks are small, well-scoped, and single-turn by default.** That assumption does real work and is worth stating up front, because it dissolves several problems rather than solving them.

A task sized to one agent turn needs no session resume, so the most expensive degradation in § 6 never fires. It bounds cost error: a bad estimate on a 5k-token task is wrong by at most 5k, and estimation error across many small tasks partially cancels instead of compounding into one runaway. It makes acceptance criteria checkable, which is what makes the supervision loop terminate. And it is what INVEST's *Small* has always been for.

**Multi-turn supervision is therefore the exception, not the design center.** It exists for tasks that genuinely cannot be decomposed further, and `dacli` should treat reaching turn 3 as a signal that the task was mis-sized rather than as normal operation.

There is a floor, though, and it is easy to miss: **the brief is a fixed cost per task.** Goal chain, scope boundary, constraints, glossary, and shortcut catalog run 1–2k tokens before the agent does anything. Slice work finer than a few multiples of that and you pay the brief tax N times for a job that is mostly overhead. So task size is bounded on *both* sides — above by what fits one turn with room to work, below by the brief. The sweet spot is roughly 3–10× the brief cost, and `dacli lint` should warn outside it in either direction.

## 2. Two integration planes

Every runtime is integrated on two independent planes, and confusing them is the main way this design could go wrong.

**The control plane — `dacli` → CLI.** How `dacli` launches the process: binary, flags, prompt delivery, sandbox settings, session resume, exit-code semantics. This is per-runtime and messy, and it is what an adapter encapsulates.

**The data plane — CLI → workspace.** How the child's *work* comes back. This is identical for every runtime, because the child writes to the workspace through `dacli` itself, exactly as it does today.

The adapter absorbs all the heterogeneity. Nothing downstream of the data plane knows or cares which CLI produced a finding.

## 3. The return channel is the workspace, not stdout

The tempting design is to parse each CLI's output for results, and every runtime having a different output format then becomes the central engineering problem — a parser per vendor, rewritten each time one ships a new version.

Don't. **Structured results come back through the workspace; stdout is only a transcript.**

The child reports by calling `dacli` (or its MCP tools): `dacli ask`, appending a `finding` event, checking an acceptance box, `dacli task done`. Those writes are already atomic, attributed, append-only, and contention-free. The architecture for collecting results from many concurrent agents was built in the first version of this design; it needs no vendor-specific anything.

This buys four things:

- **Format independence.** A vendor changing its JSON schema breaks a progress bar, not the results.
- **Partial-failure survival.** An agent killed at its budget ceiling still leaves behind every finding it wrote before dying. Parsing a final report loses everything when there is no final report — which is exactly when you most want the partial work.
- **Uniform attribution.** Every write carries the agent id regardless of which binary produced it.
- **Streaming visibility.** A parent watching `dacli events tail` sees a child's progress live, without a vendor-specific streaming parser.

stdout/stderr are still captured (§ 13), but as a transcript for debugging — never as the source of truth for what the agent found.

**The one exception**: exit status. A non-zero exit is a real signal and is interpreted per-runtime, since exit-code conventions differ.

## 4. Adapters are data, not code

A compiled-in adapter per vendor would be permanently out of date. This landscape changes monthly; flags get renamed; new CLIs appear.

So adapters are declarative files in the workspace, consistent with everything else here:

`.dacli/runtimes/<name>.md`

```markdown
---
id: rt-claude-code
kind: runtime
name: claude-code
binary: claude
detect: { version_flag: "--version", min_version: "" }

invoke:
  prompt: { mode: arg, flag: "-p" }      # arg | stdin | file
  args: ["--output-format", "json"]
  cwd: repo_root

capabilities:
  resume: { supported: true, flag: "--resume", id_from: "session_id" }
  structured_output: { supported: true, format: json }
  usage_reporting: { supported: true, path: "usage" }
  sandbox_readonly: { supported: true, args: ["--permission-mode", "plan"] }
  mcp: { supported: true, config_flag: "--mcp-config" }
  model_select: { supported: true, flag: "--model" }
  skills:                                   # see docs/SKILLS.md
    native: { supported: true, dir: ".claude/skills" }
    context_file: { supported: false }

exit_codes:
  0: ok
  default: failed

env_passthrough: [ANTHROPIC_API_KEY]
---

Notes on this runtime's quirks, discovered the hard way. This body is the
part worth writing.
```

**The flags in every shipped adapter are assumptions, not facts.** They are verified per-install by probing (§ 5), never trusted because a doc said so. I have deliberately not asserted exact flag sets for CLIs I cannot verify from here; the shipped adapters are starting points to be corrected by `dacli runtime doctor` on a machine where the binary exists.

Adapters ship for: `claude-code`, `codex`, `gemini-cli`, `opencode`, and `generic-exec` (a lowest-common-denominator adapter: one-shot, prompt on stdin, no resume, no usage reporting — enough to drive nearly anything).

A sixth ships for testing: **`mock`** — `generic-exec` pointed at a fixture script that plays an agent (reads the brief, writes scripted events, exits with a scripted code). Zero API calls, zero cost, fully deterministic. This is the entire CI story for the supervision loop, the failure taxonomy, and the budget accounting; without it, every L5 test either costs money or tests nothing.

> **One open item:** you mentioned "agy cli" and I don't know which tool that is — I'd be guessing between Amp, Aider, and Auggie, and guessing wrong would put fabricated flags in a spec. Tell me which and I'll add the adapter.

## 5. Capabilities are probed, not assumed

`dacli runtime doctor` verifies each adapter against the installed binary and writes the result to a per-machine cache (never committed — capabilities are a property of the install, not the project):

```
$ dacli runtime doctor
claude-code   ✓ binary  ✓ version 2.x  ✓ json  ✓ resume  ✓ usage  ✓ sandbox  ✓ mcp
codex         ✓ binary  ✓ version 1.x  ✓ json  ✗ resume  ? usage  ✓ sandbox  ✓ mcp
gemini-cli    ✓ binary  ✓ version 0.x  ✓ json  ✗ resume  ✗ usage  ? sandbox  ✓ mcp
opencode      ✗ binary not found on PATH
```

Three states, and the distinction matters: `✓` probed working, `✗` probed absent, `?` unprobeable (the capability exists but can't be verified without a paid call — treated as absent for planning, and reported as unknown rather than claimed).

Probes must be free or nearly so: `--version`, `--help` parsing, and at most one trivial prompt against the cheapest model. A doctor run that costs real money will not be run.

## 6. Degradation is explicit and announced

When a runtime lacks a capability, `dacli` degrades — and says so. It never silently substitutes a worse strategy.

| Missing | Degradation | Cost, stated plainly |
|---|---|---|
| `resume` | Re-send the full brief plus a turn summary each turn | Turn *n* costs roughly *n*× the context. **Mostly moot under the small-task assumption** (§ 1): single-turn tasks never resume, so a runtime without it is fully usable for the common case. It only bites on genuinely indecomposable work, where `dacli` warns at turn 3 and refuses past `--max-turns`. |
| `structured_output` | Transcript only | No usage parsing, no session id. Results still arrive (§ 3), so this degrades observability, not correctness. |
| `usage_reporting` | Estimate from transcript length | Budget enforcement becomes approximate. `dacli` marks the run's cost figures as estimated everywhere they appear, so nobody builds a decision on a number that was guessed. |
| `sandbox_readonly` | **Refuse to spawn a read-only agent** | This one does not degrade. See § 8. |
| `mcp` | Child uses the `dacli` CLI instead of MCP tools | Slightly more token overhead, no loss of function. |
| `model_select` | Use the runtime's default | Role-level model routing silently stops working, so it is reported at spawn rather than discovered later. |
| `skills.native` | Compile to the runtime's context file; failing that, inline into the brief | Progressive disclosure is lost — the skill's full body becomes an every-turn token tax, stated at spawn. A skill whose `min_delivery` can't be met is omitted **and announced**. ([SKILLS.md](SKILLS.md)) |

The rule: degrade observability and cost silently-ish, never degrade a safety property.

## 7. The supervision loop

"Chatting with the CLI" is a multi-turn conversation between a parent and a child it spawned. The protocol:

```
1. Parent assembles the brief          dacli context <task> --budget N
2. Parent spawns the child             runtime.invoke(brief + role skills + shortcut catalog)
3. Child works, writing to the workspace as it goes
4. Child exits (or hits a turn cap)
5. Parent evaluates:
     - acceptance boxes checked?
     - lint clean?           dacli lint <task>
     - verification shortcut passing?   dacli run test
6. If satisfied  → accept, sync events, done
   If not        → send a targeted correction as the next turn, go to 3
   If out of budget or turns → stop, record what happened, escalate
```

Two properties make this loop terminate, which is the whole difference between this and an open-ended conversation:

- **Every turn is evaluated against a fixed, external criterion** — the task's acceptance criteria, written before the work started. Not the parent's impression of whether the answer sounded good.
- **Turns and budget are both capped**, and exhausting either is a recorded outcome (`stalled`), not an error to be retried into oblivion.

### Why this isn't the chat room I argued against

In [TEAM.md § 3](TEAM.md) I rejected agent-to-agent chat. This is conversation between agents, so the distinction has to be real or that argument was reflexive.

It is real, and it is the completion criterion. My objections were: agents are agreeable, so peer discussion converges without adding information; and a conversation has no definition of done.

Neither applies here. This is **supervised delegation, not peer discussion.** The parent is not seeking the child's agreement — it is checking work against acceptance boxes it wrote in advance. Agreeableness is not a failure mode when one side is running a test suite. And the loop terminates the moment the criteria are met, or the budget runs out, whichever comes first.

The test for whether a conversation belongs in `dacli`: **is there something outside the conversation that decides when it ends?** Acceptance criteria, yes. Two agents talking about architecture, no.

## 8. Permissions: the caveat that goes away

DESIGN.md § 6 concedes that `dacli`'s capability system is cooperative — a subagent with shell access can edit the workspace markdown directly and bypass `dacli` entirely. That has been the least comfortable claim in the design.

**Spawning changes it.** When `dacli` launches the child process, it controls the child's sandbox flags. A read-only agent gets launched with the runtime's own read-only or plan mode, which the child cannot escape by choosing not to call `dacli` — enforcement moves from convention into the runtime's process boundary, where it can actually be enforced.

This is a real upgrade and it comes with sharp limits, all of which have to be stated:

- **Only for agents `dacli` spawned.** An agent that runs `dacli` from a shell somebody else started is exactly as cooperative as before.
- **Only as good as the runtime's sandbox.** `dacli` is trusting each vendor's implementation, and those vary in strength and in what they actually cover.
- **A missing sandbox is a refusal, not a degradation.** If a runtime cannot enforce read-only, `dacli` refuses to spawn a read-only agent on it rather than spawning an unrestricted one and labeling it `ro`. A capability that silently isn't there is worse than one that was never claimed.

Grant resolution order, tightest wins: **parent's grant** (attenuation, § DESIGN 6) → **role's grant** (a ceiling request) → **runtime's enforceable maximum** → **`--grant` at spawn**. No layer can widen what a tighter layer allows.

## 9. Budget and cost

Every run carries a token budget. The parent's budget bounds the sum of its children's — a tree cannot spend more than its root was given.

Where `usage_reporting` exists, actuals are read from the runtime. Where it doesn't, `dacli` estimates and **marks every derived figure as estimated**, because a velocity chart built on guessed numbers that doesn't say so is worse than no chart.

This is also where the SPM layer's Tier-2 reinterpretation stops being theoretical: velocity as tasks per 100k tokens, burndown against spend, token boxes instead of time boxes ([SPM.md](SPM.md) § Tier 2) all need real per-run cost, and spawning is the first point where `dacli` can actually observe it.

Budget exhaustion terminates a run. It is recorded as `stalled` with everything the child wrote before dying still intact — which is § 3's payoff.

## 10. Heterogeneity as verification diversity

The strongest argument for supporting many runtimes is not vendor-neutrality or price. It is that **a verification panel drawn from one model is a single point of failure wearing several hats.**

In TEAM.md I noted that the one real agent analogue of Groupthink is a panel of verifiers agreeing because they share prompts and context. Prompt diversity helps. Running the verifiers on *different vendors' models* helps more, because the failure modes are genuinely uncorrelated — different training data, different post-training, different blind spots.

```
dacli verify <task> --panel claude-code,codex,gemini-cli --require 2
```

A finding confirmed by two of three different-vendor models is meaningfully stronger evidence than one confirmed by three samples of the same model, and no single-vendor tool can offer it. `dacli doctor` should therefore flag a verification panel that ran entirely on one runtime.

Verdicts are recorded, not just counted: each panelist appends a `finding` event carrying `verdict: confirmed | refuted` plus its reasoning, and the `--require N` tally is **derived from the log** — same rule as shortcut `uses`. This keeps a panel auditable after the fact (which panelist refuted, on what grounds, on which runtime) instead of collapsing three judgments into one unexplained integer.

Role-level routing follows the same idea: an `implementer` role and a `reviewer` role should default to different runtimes, so review is not the author grading its own homework with the same instincts.

## 11. Failure taxonomy

Each failure gets a distinct recorded outcome, because "it failed" is useless to the agent reading the log afterward.

| Failure | Detection | Response |
|---|---|---|
| Binary missing | Probe at spawn | Refuse; suggest an available runtime |
| Auth missing/expired | Probe, or exit-code mapping | Refuse; never prompt for a credential, never read one from the workspace |
| Rate limited | Exit code / stderr pattern | Retry with backoff, capped; then fall back to another runtime if the role permits |
| Transient API error | Exit code | Bounded retry — the one place retry is legitimate |
| Non-zero exit, work done | Exit code + events written | Record `partial`; the events are still valid |
| Hang | No events and no output past a deadline | Terminate, record `stalled` |
| Budget exceeded | Usage tracking | Terminate, record `stalled` |
| Turn cap reached | Loop counter | Stop, record `stalled`, escalate per § TEAM 3 |
| Runaway spawning | Depth and count caps | Refuse; this is the one that can cost real money fastest |

**Depth and fan-out caps are mandatory, not optional.** A tree of agents that can spawn agents needs a hard ceiling, defaulting low (depth 3, 16 concurrent). The Kanban WIP limit from TEAM.md constrains one role; this constrains the whole tree.

## 12. Security

- **Credentials never touch the workspace.** Adapters declare `env_passthrough` by variable *name*; values come from the environment. A workspace that is committed to git must never be able to leak a key.
- **Transcripts are untrusted input.** A child's output can contain anything the model was fed, including text from a file that tries to instruct the parent. Transcripts are data. A parent must never execute or obey instructions found in a child's transcript, and dacli should never surface transcript content as if it were a directive.
- **Shortcuts remain gated per § SHORTCUTS.** A spawned child inherits its role's toolkit, not the parent's.
- **Prompt injection crosses the tree.** A child that reads a hostile file and writes its content into a `finding` puts that content into every sibling's brief. Findings authored by agents should be attributed and, where a runtime supports it, sanitized of instruction-shaped text. This is an unsolved problem and § 17 says so.
- **`--dangerously-*` style flags are never set by `dacli`.** If a runtime needs one, that is a human's decision, made in the adapter file, in a commit.

## 13. Transcripts and reproducibility

Each run records to `.dacli/runs/<run-id>/`:

```
invocation.json     exact argv, env var names (never values), cwd, adapter version
transcript.log      captured stdout/stderr
usage.json          tokens and cost, flagged if estimated
outcome.md          ok | partial | stalled | failed, with the reason
```

`.dacli/runs/` is **gitignored by default.** Transcripts are large and can contain repository content that was fine in a working tree and is not fine in a pushed branch. `dacli runs prune` bounds growth.

`invocation.json` makes a run reproducible: the same adapter, same flags, same brief. Given a non-deterministic model this reproduces the *setup*, not the output — which is still most of what you need when debugging why an agent did something strange.

## 14. Roles select runtimes

Roles gain optional runtime routing:

```yaml
runtime: claude-code          # or a preference list
model: default                # adapter-resolved
fallback: [codex, gemini-cli] # on rate limit or unavailability
max_turns: 6
budget: 40000
```

Resolution order: `--runtime` at spawn → role's `runtime` → workspace default → the only available one. Unavailable runtimes fall through to `fallback` only if the role declares it — a silent vendor switch would undermine § 10's whole point, since a panel that quietly collapsed onto one runtime looks diverse and isn't.

## 15. Commands

| Command | Purpose |
|---|---|
| `dacli runtime list` | Configured runtimes and probed capabilities |
| `dacli runtime doctor` | Probe installs, verify adapter assumptions, cache results |
| `dacli runtime add` | Add an adapter |
| `dacli spawn --runtime X --task T` | Spawn a child on a runtime (extends TEAM.md § 4) |
| `dacli supervise <task>` | Run the § 7 loop to completion or budget |
| `dacli verify <task> --panel a,b,c` | Multi-runtime verification panel |
| `dacli runs list\|show\|prune` | Run records |

## 16. Slash commands as work units

Every one of these CLIs ships slash commands — `/review`, `/commit`, and whatever the repo defines in its own commands or skills directory. `dacli` should invoke them rather than reinventing them.

**The argument is the same one as shortcuts, one level up.** A shortcut memoizes a *shell* incantation somebody paid to get right. A slash command reuses *prompt engineering* that a vendor maintains, tests against their own model, and updates when the model changes. A review prompt `dacli` invents will be worse than `/review` and will rot faster. Three tiers of reuse, and they compose:

| Tier | Reuses | Maintained by |
|---|---|---|
| Shortcut | A shell command | This project |
| Slash command | A prompt | The CLI vendor, or the repo |
| Skill | Role knowledge loaded at spawn | The workspace, compiled per runtime ([SKILLS.md](SKILLS.md)) |

### Invocation

For most CLIs a slash command is just a prompt that begins with `/`, so the adapter capability is small:

```yaml
capabilities:
  slash_commands:
    supported: true
    invoke: prompt_prefix        # prompt_prefix | flag | unsupported
    list: "--help"               # how to enumerate, where possible
```

A task can name one directly:

```yaml
command: review                  # an abstract verb, not a literal string
```

### Portability is shallow, and the mapping is the honest part

`/review` on two different CLIs is not the same prompt, and on a third it may not exist. So tasks name **abstract verbs**, and each adapter maps verb → its own literal command:

```yaml
verbs:
  review: "/review"
  commit: "/commit"
  explain: "/explain"
```

An unmapped verb falls back to a plain prompt, announced. `dacli runtime doctor` enumerates available commands where the CLI can list them, so an adapter claiming a verb the binary does not have is caught by probing rather than at 2am.

Three limits worth stating rather than discovering:

- **Same name, different behavior.** Cross-runtime verb mapping gives portability of *intent*, not of output. A verification panel (§ 12) spanning runtimes should expect different shapes of answer, which is exactly the diversity it wants — but it means results must come back through the workspace (§ 3), not by parsing.
- **Some slash commands assume an interactive session** and behave differently or not at all in one-shot mode. Probed, not assumed.
- **Repo-defined commands are project state.** A `/deploy` defined in the repo is as powerful as a destructive shortcut and deserves the same gating, but `dacli` cannot see inside it. Slash commands are therefore treated as **write-effect by default** unless the adapter declares a verb read-only.

Slash commands are good for well-known verbs at the edges of a task — review, commit, explain. Core task execution should stay a plain brief, because that is the part `dacli` can actually reason about, gate, and check against acceptance criteria.

## 17. The non-goal, restated honestly

DESIGN.md § 2 said "not a job runner: no process execution, retries, or timeouts." Shortcuts already narrowed that once. This narrows it again — spawning means processes, timeouts, and bounded retry.

**Two erosions is where you stop patching and restate the boundary.** The original wording is no longer describing the tool. What is actually true:

> `dacli` supervises **agent processes it spawned**, one per task, each bounded by a budget, a turn cap, and a timeout. It is not a workflow engine: it owns no DAG of jobs, schedules nothing against a clock, and never decides that arbitrary work should run. Queues remain ordered step lists with a cursor, executed by the agent.

The line that survived both erosions, and the one to defend: **`dacli` runs agents, not work.** The moment it grows a cron trigger, a job dependency graph, or a step that isn't an agent or a named shortcut, it has become a CI system with a markdown skin, and it should stop.

## 18. Open questions

The small-task assumption (§ 1) retired or shrank three of the five questions this doc originally carried. What it did and did not fix, kept explicit because the difference matters:

**Dissolved.** *Turn-summary compression for non-resuming runtimes.* Single-turn tasks never resume, so the compression step — itself a model call, with its own cost and its own habit of dropping the constraint that mattered — simply does not happen. Closed.

**Substantially reduced.** *Cost estimation without `usage_reporting`.* Transcript-length heuristics are still wrong by large factors, but the consequence of a bad estimate is bounded by the task, and errors across many small tasks partially cancel rather than compounding into one runaway. Wide error bars on a small number are survivable. They remain labeled as estimates everywhere they appear.

**Easier to answer.** *Is `verify --panel` worth the multiple of cost?* Panels over small tasks are cheap enough to run as an experiment rather than argued about. Still unmeasured, still should be measured before it becomes a default.

**Unchanged.** *Adapter drift.* Vendors rename flags; `runtime doctor` catches it at spawn time, which is late. A pinned-version matrix in CI would catch it earlier at the cost of infrastructure this project has avoided.

**Not fixed, and the shape changed for the worse.** *Cross-tree prompt injection* (§ 12). Small tasks give each child less context to leak and a narrower scope to abuse, and narrow acceptance criteria make a poisoned result more likely to be caught. But there are now *more* children, each reading files, each able to write a finding that enters every sibling's brief — so the per-child blast radius shrank while the propagation surface grew. Enabling inbound GitHub sync on a public repo ([GITHUB.md § 7](GITHUB.md)) widens the door further, since anyone who can comment can put text into an agent's context.

This remains the most serious unresolved problem in the design. Attribution supports an audit after the fact; it prevents nothing. It should not be smoothed over in a README.

Two questions the templates work added:

6. **Continuous flow versus stage gates.** Small tasks want to flow; gates want batch checkpoints. Kanban and UP are genuinely different philosophies and this design currently ships both without saying which wins when they conflict ([TEMPLATES.md § 9](TEMPLATES.md)).
7. **Gate evaluation cost.** Placeholder and ambiguity checks are local and free; `shortcut: test` and `lint: clean` are not. Caching against content hashes is the likely answer.
