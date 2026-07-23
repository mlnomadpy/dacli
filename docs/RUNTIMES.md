# Runtimes: driving coding-agent CLIs

**Status: §§ 1–18 are the original design. Much of it now ships.** The spawn
lifecycle, permission sandboxing, run records, the supervision loop,
verification panels, token calibration, and the integration tail are all
implemented. **[Part II](#part-ii--implemented-reference) (§§ 19–23) documents
the surface as it actually exists** in `internal/features/execution`,
`internal/features/vcs`, and `internal/store` — it is the reference to trust
where the spec above and the code disagree.

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

---

# Part II — Implemented reference

The spec above argues for a design. This part documents what runs today, flag
by flag, verified against the source. It is deliberately literal: where the
spec says "budget bounds the tree" and the code records but does not enforce a
budget, this part says so.

## 19. The spawn command and its gates

`dacli spawn` mints a child identity, freezes the brief, applies the sandbox,
runs the child to completion (or backgrounds it), and writes a run record. Its
full flag surface:

```
dacli spawn --task <ref> [--runtime name] [--role r] [--grant ro|rw] [--model m]
            [--worktree] [--detach] [--claim path,path] [--pr]
            [--review [--pr-number N]] [--budget N] [--max-tokens N]
            [--timeout sec] [--cooperative] [--advise] [--force]
```

| Flag | Effect |
|---|---|
| `--task <ref>` | Required. Resolves a task by seq, ULID, or slug. |
| `--runtime name` | Which adapter to launch. Falls back to the role's `runtime:`; with neither, spawn refuses. |
| `--role r` | Seeds grant, runtime, and model defaults, and applies the role's WIP limit, seniority gate, and phase gate. |
| `--grant ro\|rw` | Read-only or read-write. Defaults to the role's grant, then to `ro`. |
| `--model m` | Model tier (adapter maps it to its `--model` flag). Used for cost routing — reviewer=opus, junior=haiku. |
| `--worktree` | Isolate the child in its own git worktree on branch `dacli/NNN-slug` so parallel children never clobber each other's tree. Its `.dacli` state still redirects to the shared root, so it self-commits and self-reports there. |
| `--detach` | Start the child, print its run-id, and return immediately. Its outcome is finalized later by `dacli wait`. |
| `--claim path,path` | Declare the paths this agent will edit. If a **live** agent already claims an overlapping tree, spawn refuses (the disjointness that keeps parallel branches merge-clean). Also stamped into the run record so `dacli commit` can enforce claim-scoped staging. |
| `--pr` | Tell an `rw` child (via the `git_workflow` prompt) to open a PR for its branch. |
| `--review [--pr-number N]` | Append the `review_workflow` prompt so the child reviews a branch/PR. `--pr-number` is the PR to review; the search key is the task's branch name. **This is the only command that reads `--pr-number`.** |
| `--budget N` | A token budget, **recorded in the run record, not enforced** (the invocation line says so explicitly: "recorded, not enforced: runtime reports no usage"). |
| `--max-tokens N` | A spawn-time cost gate — see § 23. |
| `--timeout sec` | Wall-clock deadline for the child turn (default 300s). |
| `--cooperative` | Accept convention-only read-only on a runtime that can't enforce a sandbox, instead of refusing. Also bypasses the taint gate. |
| `--advise` | Print a calibrated sizing and taint status for this spawn, then continue unchanged — see § 23. |
| `--force` | Override the `--max-tokens` gate and the taint gate (loud, on stderr). |

### Spawn-time gates, in order

Spawn runs these checks; any of them can refuse before the child launches:

1. **Role gates** — WIP limit (refused if the role is at capacity), seniority, and phase. (`cmdSpawn`, `execution.go`)
2. **Runtime resolution** — a runtime is mandatory, and its binary must be on `PATH` or spawn errors with a `runtime doctor` hint.
3. **`--max-tokens` cost gate** (§ 23) — refuses (exit 3) when the band's measured token cost exceeds `N`, unless `--force`; below `n≥10` it warns instead of refusing.
4. **Taint gate** — if the task's brief sits in an external source's blast radius (`store.Taint("external:")`), refuse (exit 3) rather than feed a possibly-injected brief to a fresh child. `--force` or `--cooperative` overrides. This is § 18's cross-tree injection turned from an audit query into a gate at the point of consumption.
5. **Sandbox gate** — for an `ro` grant, the runtime must declare a read-only arg set (`SandboxRO`), or spawn refuses with *"spawning an unrestricted process labeled ro would be a lie"* — unless `--cooperative`. This is § 8's "a missing sandbox is a refusal, not a degradation," enforced.
6. **Claim conflict** — `--claim` paths that overlap a live agent's claim refuse.

Only then is the identity minted, the claim stamped (`claimed by <childID>` in the task Log — the span start calibration reads), the brief assembled and frozen to `brief.md`, and the process run.

### Outcome

A foreground spawn evaluates the child against the fixed criterion — acceptance
boxes checked plus events the child actually wrote to the workspace — and
records one of: `ok`, `partial` (the run errored but the child wrote events),
`failed` (errored, nothing written), or `stalled` (timed out). Partial work
survives a dead child; that is the workspace-return-channel payoff of § 3. The
run record (`.dacli/runs/<run-id>/`) holds `brief.md`, `invocation.txt`,
`outcome.md`, `transcript.log`, and — for a usage-reporting runtime —
`usage.txt`.

## 20. The agent lifecycle commands

Once spawned (especially `--detach`ed), a child is managed through these:

| Command | Purpose |
|---|---|
| `dacli wait [run…]` | Block until the named detached run(s) finish — or all live agents if none named — then **finalize each outcome from the workspace effects it left behind** (`finalizeRun`). This is where a detached stream-json run's `usage.txt` is harvested (§ 23). Flags: `--interval sec` (poll cadence, default 3s), `--timeout sec` (overall cap, default 3600s). |
| `dacli agents` | List agents whose process tree is still alive, with the whole group's RAM, CPU, GPU, proc count, and uptime. Liveness is probed live (and PID-identity-checked), so an exited agent simply doesn't appear. |
| `dacli agents --tail` | Under each agent, print its **last non-empty transcript line** — its current activity. RAM/CPU can't tell a reasoning agent from a wedged one; a live tail can (a thinking agent's last line keeps moving). |
| `dacli agents --max-rss 2G --max-runtime 15m [--reap]` | Flag agents over a RAM or runtime budget as runaways; with `--reap`, kill the whole over-budget tree. |
| `dacli logs <ref> [-f] [--tail N]` | Print, or with `-f` follow, a run's transcript. A detached child streams straight to the transcript file, so `-f` tails it like `tail -f`. Detached **stream-json** runs write raw JSON to the transcript, so `logs` renders each event to readable text on read. |
| `dacli kill <ref> \| --all [--grace sec]` | Terminate an agent and its **entire process group** — SIGTERM, then SIGKILL after a grace window (default 3s) if anything survives — so no orphaned children are left holding resources. Writes a `killed.txt` audit crumb. |
| `dacli runs list \| show <ref> \| prune [--keep N]` | The recorded run archive (newest first; `prune` keeps 20 by default). |
| `dacli supervise --task <ref>` | The § 7 loop: spawn → evaluate against acceptance boxes → send a targeted correction as the next turn → repeat, until accepted or `--max-turns` (default 3). One child identity owns the task across turns; turn 3 prints a "this task should be decomposed" note. Applies the child's events between turns. |

`dacli agents`, `kill`, `wait`, and `logs` all resolve a run by run-id prefix or child-id, and all funnel liveness through the same PID-identity check, so a run whose PID was recycled by an unrelated process is filtered out before any sample or kill.

## 21. The integration tail

Work comes back on branches; these land it. Ownership rule throughout:
**box-checking and task-closing are owner-only** — a read-only child that runs
`dacli accept` records a *proposal* the owner later applies.

| Command | Purpose |
|---|---|
| `dacli commit "<msg>" --task NNN` | Commit **in the agent's own worktree**, authored as `agent (role)` with `Dacli-Agent`/`Dacli-Role`/`Dacli-Task` trailers. Refuses to commit on `main`/`master`. Stages `git add -A` unless `--no-add`. **Enforces claim scope**: refuses code files staged outside the agent's recorded `--claim` unless `--force` (`.dacli/` is always allowed). |
| `dacli accept <ref> [--verify "cmd"] [--force]` | Owner step: run the optional `--verify` command (`sh -c`, non-zero exit refuses the close), apply any pending proposals, check **every** acceptance box, and close the task — stamping `completed by` (the calibration span end). `--all` accepts every task with a pending proposal in one pass, gating the whole batch once with `--verify`. `--force` (root only) reconciles a task orphaned by a finished spawned agent — one that will never run `sync` again to apply its own proposal — by adopting ownership before closing; with `--all`, it applies that same override to every orphaned task in the batch, not just root-owned ones. |
| `dacli integrate [--tasks <refs>] [--into <branch>] [--project p]` | Merge task branches into `--into` (default `main`). Resolves either the explicit `--tasks` ref list (order preserved) or every done task. Serial; a clean merge removes the worktree and deletes the branch; a **conflict blocks that one task and stops — never half-merges**; a genuine non-conflict failure propagates as a non-zero error rather than being mislabeled a conflict. |
| `dacli ship [--into b] [--project p] [--verify c] [--push] [--dry-run] [--no-accept] [--no-integrate]` | The one-command wave tail: `accept --all --force` → `integrate` the resulting done branches → commit the `.dacli` record (`git add -- .dacli` only, never `-A`) → optionally `--push`. `--force` is always forwarded to `accept`, which only honors it for root — so run as root, ship auto-closes a wave's tasks left owned by agents that already finished and will never sync to apply their own proposal, instead of stalling the pipeline on an orphan. Stops at the first failing step (so it never commits a record for an integrate that didn't happen), detects a merge-conflict block semantically, and reports the count of branches **actually** merged. `--dry-run` prints the plan and executes nothing. |
| `dacli merge --task NNN [--into b]` | Merge one task's branch; conflict blocks the task and records an `EventBlock`, never half-merges. |
| `dacli pr --task NNN [--base b] [--with-verdicts]` | Open a PR via `gh`; the body carries acceptance + finding notes + `Fixes #issue`. `--with-verdicts` posts the verify panel's recorded verdicts as a `gh pr review <branch>` comment (resolved by branch, not PR number). |
| `dacli worktree add\|list\|remove`, `dacli push`, `dacli blame`, `dacli contrib` | Worktree lifecycle, branch push, per-line authorship, and the per-role/per-agent contribution + defect-rate rollup. |

## 22. Runtimes as data

The adapter is still a workspace file (§ 4). `dacli runtime add <name>` builds
one from a preset and overrides:

```
dacli runtime add claude-code --preset claude-code
dacli runtime add mycli --binary mycli --mode stdin --arg --json --env HOME \
                        --model-flag=--model --usage-format stream-json
```

Two presets ship: **`claude-code`** (binary `claude`, prompt as an `-p` arg,
read-only sandbox `--allowedTools Read,Grep,Glob,LS,Bash(dacli:*)`, model flag
`--model`, env allowlisted to `HOME PATH USER LOGNAME TMPDIR` — deliberately
**no `ANTHROPIC_API_KEY`**, so children run on the user's own Claude Code login,
never API billing) and **`generic-exec`** (no binary, prompt on stdin, no
sandbox). `dacli runtime list` shows the configured adapters; `dacli runtime
doctor` probes each binary on `PATH` and its `--version` for free, reporting a
declared-but-unprobed sandbox honestly rather than claiming it.

`--flag`, `--arg`, `--sandbox-ro-arg`, and `--model-flag` take their value
verbatim, even one starting with `-` (`--model-flag --model` works directly).
Every other flag still resolves the parser's fundamental "is the next
`--token` a value or the next flag?" ambiguity the same way Go's own `flag`
package does: the `=` form (`--key=--value`) or a literal `--` terminator
(`--key -- --value`).

## 23. Token actuals and calibration

The spec's Tier-2 reinterpretation (§ 9) — cost in tokens, not wall-clock —
now has a real data path. It is **opt-in per runtime** and leaves text runtimes
byte-for-byte unchanged.

### Opting in: `usage_format: stream-json`

An adapter's frontmatter field `usage_format` (set via `runtime add
--usage-format stream-json`) turns on machine-readable usage capture. When it
equals `stream-json`, `execRuntime` appends `--output-format stream-json
--verbose` to the child's argv (the `claude` CLI requires `--verbose` alongside
`stream-json` under `--print`). An empty `usage_format` leaves argv untouched —
a text runtime is unaffected.

### Capturing `usage.txt`

The child's stream is drained by `teeStreamJSON`, which renders each event to
readable transcript text **and** captures the terminating `result` event's
usage. On a foreground run this happens live; `writeUsage` then writes, into the
run directory beside the transcript:

```
output_tokens: <n>
input_tokens:  <n>
num_turns:     <n>
cost_usd:      <f>
```

A **detached** run streamed raw JSON straight to `transcript.log` with no live
parser (the parent had already returned), so `finalizeRun` — invoked by `dacli
wait` — re-reads the transcript afterward and harvests the same usage. This is
self-detecting: a plain-text transcript yields no `result` event, so nothing is
written. The parser uses an uncapped line reader (an over-long earlier event
can't truncate the stream before the final usage event) and surfaces a read
fault rather than silently masquerading as a text runtime.

### Bands: calibrating the agent, not the task size

D1's insight is that estimation should calibrate the **agent** that did the
work, not the task's size. A **`Band`** is `role × model × runtime`, read from
each run's `invocation.txt`. `store.LoadCalibration` walks the runs directory
once, joins each done task to its completing run's band and `usage.txt`, and
pairs the task's three-point estimate (`Te`) against its actual. One
`CalibSample` carries:

- `Hours` = the wall-clock claim→completion span (the fallback proxy) → `Ratio() = Hours/Te`, hours-per-point.
- `Tokens` = the joined `output_tokens` (0 if no usage was captured) → `TokenRatio() = Tokens/Te`, **output-tokens-per-point, the real unit**.

`store.MedianTokenRatio(samples, band)` is the shared primitive: the median
`TokenRatio` across the band's token-bearing samples, plus `n` (how many). It is
computed one way, so display and enforcement can never diverge.

### The n ≥ 10 provisional gate

Below ten samples a band's spread is noise, so `dacli calibrate` and `estimate`
gate on it:

- **`n ≥ 10`** — the band prints a `p10–p90` range and is marked
  `AUTHORITATIVE`; for tokens, *"n≥10: tokens ARE the estimate."* In `dacli
  estimate` the empirical band distribution is printed as the answer and the
  PERT three-point becomes the prior.
- **`n < 10`** — the band prints its median marked `provisional, n<10 — no
  calibrated range`, and no calibrated number is offered. Below ten overall,
  briefs stay silent about calibration.

`dacli calibrate` shows three views: size bands, agent bands by wall-clock, and
agent bands by **tokens/point — PREFERRED** (with the honest caveat that
wall-clock is only the fallback for runs without usage).

### `--advise` and `--max-tokens`: acting on the log at spawn

Both read the same `MedianTokenRatio × Te`, so the number you're *shown* and the
number that's *enforced* are identical:

- **`spawn --advise`** (`printAdvisory`) — display only. With a token-bearing
  band at `n ≥ 10` and an estimated task, it prints the suggested budget
  `MedianTokenRatio × Te` output tokens, labelled measured-token-cost. At `n <
  10` it prints `PROVISIONAL` with no firm number. With no token history it
  falls back to the wall-clock proxy under the same `n ≥ 10` gate. It also
  prints the task's taint status. The spawn then proceeds unchanged — advice
  never decides (axiom 3).
- **`spawn --max-tokens N`** (`bandTokenBudget`) — enforcement. `expected =
  MedianTokenRatio × Te`. If `expected > N` the spawn **refuses (exit 3)** citing
  the calibrated sample count, unless `--force`. Below `n < 10` the estimate is
  provisional, so it **warns and spawns anyway** rather than hard-refusing on
  thin data. A band with no token history (a text runtime) or an unestimated
  task has nothing to enforce honestly, so it proceeds with a note.
