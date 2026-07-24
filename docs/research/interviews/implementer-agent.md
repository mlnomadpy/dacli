# Interview — Implementer agent: operational needs & steering signals

Status: **filled research instrument.** This is the implementer-agent segment's answers against
[`../INTERVIEW_GUIDE.md`](../INTERVIEW_GUIDE.md) §5 (transcript study) and §6 (prompted self-report) —
not a spec and not a backlog. It confirms or refutes the guide's hypotheses *from the agent's side of
the loop* and feeds [`../../PROPOSALS.md`](../../PROPOSALS.md); it does not amend
[`DESIGN.md`](../../../DESIGN.md) or [`ARCHITECTURE.md`](../../ARCHITECTURE.md).

**POV note.** The guide is explicit that agents are not interviewed live — they are studied via transcript
review and prompted self-report against real runs ([`../INTERVIEW_GUIDE.md`](../INTERVIEW_GUIDE.md) line 26).
This document is that self-report, authored from inside a real implementer run and grounded in the mechanics
the agent actually operates under, cited `file:line` the way a finding is. Impressions don't survive a
`verify` panel and neither should these; every claim below points at code.

**Trust floor.** Read every answer here as a single-source, `unverified` lead, exactly as the workspace
grades a fresh finding (`internal/prompts/tpl/protocol_preamble.md:27`). A need is `confirmed` only when it
recurs across ≥3 independent transcripts ([`../INTERVIEW_GUIDE.md`](../INTERVIEW_GUIDE.md) §9). One run is a
lead, not a fact.

---

## 0. The one thing to understand first: the brief *is* the agent's dashboard

The human operator's observability surface is the dashboard; the implementer agent's observability surface
is **the brief** — the context document assembled by `dacli context`
(`internal/brief/brief.go:57` `Assemble`; the package doc at `brief.go:1-8` calls it "the product...
everything else exists so this function has something to slice"). Every question in the guide about *what
the agent needs to see* is a question about **what belongs in the brief and when it refreshes**, because the
agent never renders the dashboard — the dashboard is read-only for humans (`internal/features/dashboard/dashboard.go:110-112`,
`GET /api/state` only encodes JSON, never mutates).

This reframes the whole steering conversation. A steering signal that lands on a surface the headless agent
cannot read is not steering — it is theater the operator mistakes for a correction. The agent's only inputs
are (a) its context/brief and (b) OS signals. Anything meant to reach the agent must arrive on one of those,
at a turn boundary, or it does not arrive at all.

---

## 1. What context & steering *helps* (RQ1, RQ2)

The brief already carries most of what steers a run well, and the highest-value pieces are the ones that
suppress [`DESIGN.md`](../../../DESIGN.md) §1's three failure modes:

| What the brief gives me | Where | Why it steers me |
|---|---|---|
| **Out-of-scope list** | `brief.go:110` | Keeps me inside my claim; the single cheapest way to stop me editing a sibling's tree. |
| **Constraints + prior decisions** (what was chosen, what was *rejected*, and the `--because`) | `brief.go:120-148` | The highest-signal context there is. Seeing a rejected alternative means I never re-propose it — this is the direct antidote to relitigation. |
| **What siblings found**, carrying a trust floor | `brief.go:208-288` | The antidote to sibling blindness — I don't burn budget re-discovering what a sibling already learned, and the floor (`brief.go:271-285`) tells me how far to trust it. |
| **Calibration line** (my expected token/size band) | `brief.go:73-85` | Tells me how big this task is *supposed* to be, so I can right-size scope up front. |
| **Codebase map + glossary** | `brief.go:116`, `175-184` | Orients me in one read instead of a fan-out crawl. |

**The load-bearing insight for steering:** the best mid-run correction is *a better brief given once at spawn*,
not a message injected later. Prior decisions (`brief.go:120-148`) are pre-emptive steering — they correct a
wrong turn before I take it. Everything the guide files under "steer a live agent" is, from my side, mostly a
symptom of a brief that didn't say the thing up front. Probe whether a confirmed steering pain would have been
eliminated by a constraint the brief could have carried — most are ([`../INTERVIEW_GUIDE.md`](../INTERVIEW_GUIDE.md)
RQ2's own "comfort blanket vs real need" question).

### Where the brief leaves me blind (observability gaps, RQ1/RQ5)

- **The brief is a snapshot, frozen at spawn.** `--record` freezes it for replay
  (`internal/features/briefing/briefing.go:42-55`). Sibling findings that land *after* I spawn never reach me.
  A sibling who refutes a finding I'm building on, or claims a path I'm about to touch, is invisible until my
  next full context refresh — which, mid-task, I usually don't do.
- **Working-memory cap trims the tail.** `MillerCap = 7` (`brief.go:48`) caps constraints, risks, findings,
  and lessons; `trim()` drops droppable sections from the *bottom* under budget (`brief.go:422-452`). The 8th
  sibling finding — possibly the one that would have saved my run — is simply not there. This is a feature
  (working memory is finite), but it means "just add it to the brief" is never free.
- **No live sibling state.** "What siblings found" is *reported* findings only. What a sibling is doing *right
  now* — in-flight, mid-edit, on which paths — has no representation anywhere I can read.

---

## 2. When human chat / intervention *helps* vs *harms* mid-run (RQ2 → H3, the core question)

This is the sharpest question in the guide for my segment, and the answer is asymmetric.

### Interruptive, synchronous chat HARMS. This is architectural, not a preference.

The headless contract is built on *never blocking*:

> "You are running HEADLESS: no human is watching this session and no one can answer a confirmation prompt.
> **Never pause to ask permission and never wait for approval** — decide and act within your grant and sandbox...
> A blocked question means `dacli ask` (which records it) and then STOP — it does not mean wait."
> — `internal/prompts/tpl/protocol_preamble.md:9`

A chat message pushed at a live agent violates this three ways:

1. **It blocks.** If the message expects acknowledgment or a reply, the agent either stalls waiting (breaking
   the never-wait contract) or ignores it (making the feature a lie). There is no third option for a headless
   loop.
2. **It derails a coherent plan.** A correction arriving mid-reasoning — between reading a file and editing it,
   inside a multi-step tool sequence — lands as noise. The per-turn stall timeout (`internal/features/execution/execution.go:799`)
   assumes forward motion; an interruption that makes me re-plan mid-turn can trip it spuriously.
3. **It invites relitigation.** A human second-guessing an approach I already chose *and recorded as a decision*
   (`brief.go:120-148`) re-opens a closed question — the exact failure mode the decision record exists to prevent.

Naive H3-steer ("inject guidance mid-run") is the single largest throughput risk in the entire hypothesis set.

### Async, recorded, non-blocking course-correction HELPS.

The forms of intervention that help all share one property: they arrive on a channel I already read, at a
boundary I choose, without blocking me.

- **A constraint or decision appended to my context, picked up at the next refresh.** This is steering shaped
  like the thing that already works — a prior decision (`brief.go:120-148`) — just delivered later. One
  sentence ("the DAG approach was tried and refused, use the topo sort in `criticalpath.go:208`") injected as a
  constraint is worth more than a chat thread.
- **`dacli ask` — the sanctioned agent→human channel** (`internal/features/collab/collab.go:66`). Note the
  direction: *agent-initiated* blocking is healthy (I chose to block because I'm genuinely stuck); *human-initiated*
  blocking is harmful. That asymmetry is the whole rule.
- **Kill + respawn with a better brief.** Blunt, but honest: it never pretends to steer a running loop. The cost
  is the lost partial work — see §3 and §4.

**Design implication (RQ3):** if mid-run steer must exist, it should be "append to the brief's constraints and
signal a context refresh," consumed at a turn boundary — never a chat bubble delivered mid-tool-call. The
read-only-projection boundary the guide worries about is, from my side, a *safety rail*: it keeps the dashboard
from becoming a synchronous interruption source.

---

## 3. What should surface — blockers, budget, claim conflicts, being-about-to-be-cancelled (RQ1)

The guide names four things the agent should have surfaced. Here is each, from the side of the agent that
needs it, with the current gap. Note the recurring shape: each must surface **to the agent through its input
channel**, not only to the operator through a UI the agent can't see.

### 3.1 Blockers — *mostly solved outbound, unsolved inbound*
When I hit a wall (a decision I can't make, a tool outside my sandbox), the protocol is `dacli ask` + STOP
(`protocol_preamble.md:9`; `collab.go:66`). My side works. The gap is twofold: (a) a blocked agent that has
STOPPED is nearly invisible to the operator unless they read transcripts — my `ask` needs to surface
*loudly* on their side, because I am now doing nothing; and (b) there is no resume-after-answer — see §6.4.

### 3.2 Budget — *the agent is blind to its own spend*
The governor tracks `windowSpent` (`internal/features/orchestration/governor.go:60,69`) and `--max-tokens`
refuses an over-band spawn *before* launch (`execution.go:294-312`). But **mid-run I have no gauge.** I learn
I overspent after the fact. I have a calibrated band (`internal/store/calibration.go:22-28`;
`internal/features/insight/insight.go:380`) that says how big this task should be — but nothing tells me *how
much of it I've burned* while I still have the chance to trim scope. A "spent vs band" line delivered into my
context at turn boundaries would let me self-govern. This is the reframe of H7 that actually reaches the agent
(§5, H7).

### 3.3 Claim conflicts — *checked at spawn, silent after*
`--claim` overlap is refused at spawn via `procmon.PathsOverlap` (`execution.go:334-342`;
`internal/procmon/procmon.go:117-131`), and worktree isolation protects my tree. But two agents on the shared
`.dacli` workspace can still collide, and a conflict that *emerges after spawn* (a sibling claims my path, or
edits files I'm mid-way through) is never signaled to me. I find out at integrate/merge time, or not at all.

### 3.4 Being about to be cancelled — *the biggest single gap on the kill path*
`dacli kill` runs `procmon.KillTree`: SIGTERM to the whole process group, poll for a grace window, then
SIGKILL survivors (`internal/procmon/procmon_unix.go:38-57`), and writes a `killed.txt` audit crumb
(`execution.go:1664,1674`). **But that grace window is for the OS to flush — it is not a signal I can use to
checkpoint.** Nothing tells the agent "you are about to be killed; commit what's salvageable and write a
finding on what you got done." A kill mid-edit therefore throws away uncommitted, salvageable work with no
chance to `dacli commit` the part that was already good. Of everything on the kill path, this is what an
implementer agent most wants changed.

---

## 4. Reaction to each hypothesis, from the agent's side (RQ4)

Reactions are the *weakest* evidence ([`../INTERVIEW_GUIDE.md`](../INTERVIEW_GUIDE.md) §7); weighted accordingly.
"Impact on me" is throughput of the implementer loop.

| # | Hypothesis | Impact on the implementer agent | Verdict |
|---|---|---|---|
| **H1** | DAG / dependency view | Neutral. The agent-facing equivalent already exists — out-of-scope (`brief.go:110`) + critical path (`internal/spm/criticalpath.go:80`). Read-only, no throughput risk, little direct value *to me*. | Operator feature; harmless. |
| **H2** | Cancel / kill from UI | **Double-edged.** Faster kill saves swarm budget on a wrong run — good. But kill *without a checkpoint grace* destroys my salvageable work (§3.4). | Fine **only** if kill = SIGTERM → *let me commit* → SIGKILL. |
| **H3** | Live transcript + chat/steer | **View half: fine** (read-only, zero agent impact). **Chat/steer half: the top throughput risk** (§2). | View yes; synchronous chat **no**; async constraint-injection only. |
| **H4** | Drag to re-prioritize | Irrelevant to me — once I'm claimed on a task I don't see backlog order. No help, no harm. | Operator feature. |
| **H5** | Pause / resume the loop | Helps the swarm *iff* pause means "governor stops spawning" (`governor.Before()` stop-file, `governor.go:111-137`) and leaves running agents alone. Pausing a *running* agent mid-tool-call = stale locks, half-written files, spurious stall timeouts. | Pause the **governor**, never the agent. |
| **H6** | Approve-gate a PR from UI | This is where my finished work *sits*. A reviewer reaches a verdict (`internal/features/execution/verify.go:159-193`) and the human gate is the bottleneck. Faster gating lands my PR sooner. | Mild positive **iff** it's an optional fast-path, not a mandatory block on `pr --auto`. |
| **H7** | Budget / token charts | As a chart I can't see: **zero value to me.** Reframed as a *spent-vs-band line in my context* (§3.2): **high value.** | Reframe for the agent, don't just chart it for the human. |
| **H8** | Retro timeline | Read-only, no agent impact. The agent-facing equivalent is already the event sample in the brief (`brief.go:290-298`). | Neutral. |

---

## 5. Features that would HARM agent throughput if added naively (acceptance-flagged)

The guide asks each segment to name where a plausible feature backfires. For the implementer agent, these are
the naive implementations that *cost* throughput:

1. **Synchronous / interruptive chat to a live agent (naive H3-steer).** Breaks the never-block headless
   contract (`protocol_preamble.md:9`); derails coherent multi-step plans; can trip the per-turn stall timeout
   (`execution.go:799`); invites relitigation of recorded decisions (`brief.go:120-148`). **The worst offender.**
2. **Kill without a checkpoint grace (naive H2).** SIGKILL mid-edit destroys uncommitted work. The grace in
   `KillTree` (`procmon_unix.go:38-57`) is currently an OS-flush window, not an agent-commit window — a kill
   button wired straight to it loses salvageable work every time.
3. **Pause that suspends a running agent, not just the governor (naive H5).** Freezing a process mid-tool-call
   risks stale locks and half-written files, and makes the stall/timeout machinery (`execution.go:799`,
   `execution.go:967-983`) misattribute a paused agent as hung.
4. **A mandatory, synchronous human-approval gate on the merge path (naive H6).** If the gate blocks
   `dacli pr --auto` rather than fast-pathing it, it converts an autonomous claim→PR→ship lifecycle into a
   human-in-the-loop bottleneck — the exact cost the swarm exists to avoid.
5. **Any steering signal delivered on a channel the agent cannot read.** A dashboard chart or chat bubble the
   headless agent never sees (the dashboard is human-only, `dashboard.go:110-112`) is worse than no signal: the
   operator *believes* they corrected the run, so the wrong run continues un-steered.
6. **Dumping live state into the brief unbounded.** The brief is capped at `MillerCap = 7` (`brief.go:48`) and
   trims from the bottom (`brief.go:422-452`). Piping raw budget/claim/sibling telemetry into it would push the
   highest-value sections (constraints, sibling findings) *out*. Any new agent-facing surface must fit the
   working-memory cap or it degrades the thing it's trying to improve.

The through-line: **the implementer agent's throughput depends on not being interrupted and not being blocked.**
Every feature above harms only when it introduces a synchronous interruption or a human-gated block into a loop
designed to run without one.

---

## 6. Answers to the §6 prompted-self-report frame

Answered against *this* run (task 140), which is itself an implementer transcript.

**6.1 — What did I know at start vs. have to discover?**
The brief gave me the task, its acceptance criteria, out-of-scope, constraints, and sibling findings. I had to
discover: (a) that my actual input document — the sibling 137 output `docs/research/INTERVIEW_GUIDE.md` —
existed and where it lived (the brief's "Existing docs" list did not name it, and did not flag that a sibling
had produced the very guide I answer against); (b) the precise `file:line` of every mechanic I cite, which took
a fan-out search. **Observability-of-brief gap:** the brief could have told me my input artifact existed. It
knew about the 137 task; it didn't connect it to mine.

**6.2 — At which turn did the run first go wrong, and what one instruction would have fixed it?**
This run didn't go wrong. Generalizing from transcript study: runs go wrong when they start *editing* before
reading out-of-scope/constraints, or when they re-propose a rejected decision. The one correction is upstream,
not mid-run: surface the single most-relevant rejected decision *first*, never let it be the one `MillerCap`
trims away (`brief.go:48,422-452`).

**6.3 — Was there a point where the right move was to stop and it wasn't taken?**
The recurring one across transcripts: when a required tool is outside the sandbox, the right move is `dacli ask`
+ STOP (`protocol_preamble.md:9`), and when a command returns "refused," the right move is to accept it as an
answer (`protocol_preamble.md:28`). Runs waste budget by *retrying a refused action* instead of treating the
refusal as terminal. Stopping earlier is often the highest-value move an agent doesn't make.

**6.4 — What would I have wanted to *ask* that I had no channel for?**
The gap is not the *ask* — `dacli ask` exists (`collab.go:66`). The gap is the **reverse channel**: there is no
resume-after-answer. When I ask, I STOP; the operator's answer arrives only as the brief of a *new* spawn, not
as a resumption of me. The missing channel is "answer reaches the blocked agent and it continues," not another
outbound question path.

**6.5 — One thing about the *other* agents I couldn't see?**
What siblings are doing **right now**. "What siblings found" (`brief.go:208-288`) is *reported* findings, frozen
at my spawn. A live view of sibling *claims* would catch the collision that the spawn-time `PathsOverlap` check
(`execution.go:334-342`) structurally cannot — because it can only check overlaps that exist *at spawn*, not
ones a sibling creates afterward.

---

## 7. Top 3 for the implementer agent (ranked by evidence & pain)

1. **A checkpoint grace on cancel.** Turn the `KillTree` grace window (`procmon_unix.go:38-57`) into a signal
   the agent can use: SIGTERM → "commit what's salvageable + write a finding on what you got done" → SIGKILL.
   *Highest pain:* every killed run today discards uncommitted, salvageable work (§3.4). Attaches to existing
   machinery; no new interruption model.
2. **A live budget-vs-band signal in the agent's context** (the reframe of H7 that reaches the agent). Expose
   my slice of `windowSpent` (`governor.go:60`) against my calibrated band (`calibration.go:22-28`,
   `insight.go:380`) at turn boundaries, so I trim scope *before* overspending, not after (§3.2). Data already
   exists; it just never reaches the one actor who could act on it in time.
3. **Async, recorded course-correction — a constraint/decision consumed at context refresh, never synchronous
   chat** (the safe half of H3). The *only* form of mid-run steering that helps rather than harms (§2).
   Delivered through the channel I already read, at a boundary I choose.

**Explicitly not in the top 3, and why:** mid-run claim-conflict detection (§3.3) is real but second-order —
spawn-time `PathsOverlap` covers most cases. DAG view (H1), retro timeline (H8), and drag-reprioritize (H4) are
operator features that are neutral to my throughput. And **synchronous chat-to-steer (naive H3) is not a
feature to build but a shape to avoid** (§5.1).

---

## 8. How this feeds forward

Per [`../INTERVIEW_GUIDE.md`](../INTERVIEW_GUIDE.md) §9: everything here is a single-source `unverified` lead.
The claims that matter most — "interruptive chat harms throughput," "kill needs a checkpoint grace," "the agent
is blind to its own budget" — graduate to [`../../PROPOSALS.md`](../../PROPOSALS.md) only when they recur across
≥3 independent implementer transcripts, each with a cited turn-ref. The write-boundary verdict the guide
demands (RQ3) is, for this segment, unusually clean: the implementer agent *wants* the read-only projection
kept, because the boundary is what stops the dashboard from becoming a synchronous interruption source. A
confirmed steering pain here argues for a **better brief and an async constraint channel**, not a chat box.

---

_Versioned with the code. Amend via PR when the mechanics cited here move — an answer that cites a stale
`file:line` is an interview run against a stale script._
