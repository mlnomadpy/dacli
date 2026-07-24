# dacli — Dashboard UX research synthesis

Status: **synthesis of discovery research.** This document rolls up the four
segment interviews behind the dashboard/steering roadmap into personas, a
human-vs-agent needs matrix, the key insights and tensions, and a prioritized,
evidence-cited feature roadmap. It is the *"where does this go next"* half of the
[interview guide's § 9](INTERVIEW_GUIDE.md#9-synthesis--evidence-discipline) —
the cross-source rollup the individual transcripts explicitly defer to. It feeds
[PROPOSALS.md](../PROPOSALS.md); it does **not** amend
[DESIGN.md](https://github.com/mlnomadpy/dacli/blob/main/DESIGN.md) or
[ARCHITECTURE.md](../ARCHITECTURE.md).

> **Trust floor: unverified.** Per the guide's evidence discipline
> ([§ 9](INTERVIEW_GUIDE.md#9-synthesis--evidence-discipline)), a need is
> `confirmed` only when it recurs across ≥3 **independent** sources, each with a
> cited quote or turn-ref. The four inputs here are one firsthand operator
> transcript ([operator.md](interviews/operator.md)) and three code-grounded
> composites / agent self-reports ([adopter.md](interviews/adopter.md),
> [implementer-agent.md](interviews/implementer-agent.md),
> [reviewer-agent.md](interviews/reviewer-agent.md)). No live human other than
> the operator has been interviewed; `docs/research/transcripts/` is empty. So
> when this document reports a need "recurs across N segments," that is
> **recurrence of the same hypothesis across composite sources** — the ranking
> signal — not yet the ≥3-real-source bar that graduates a need to a proposal.
> Read every ranking as a lead, weighted by evidence, not a verdict.

---

## 1. The four sources

| Segment | Source | Method | Voice |
|---|---|---|---|
| **Human operator** | [operator.md](interviews/operator.md) | Firsthand live interview (§ 3) | Runs the swarm daily; lives in the CLI; the dashboard is a second-monitor cockpit. **Strongest evidence** — the only firsthand human transcript, answered in past-behavior stories. |
| **Human adopter** | [adopter.md](interviews/adopter.md) | Code-grounded composite (§ 4) | "Priya," week-one adopter; judges the tool by its first hour. Reactions anchored to real first-run behavior, `file:line`. |
| **Implementer agent** | [implementer-agent.md](interviews/implementer-agent.md) | Prompted self-report against a real run (§§ 5–6) | A spawned `implementer` driving a task to a PR. The brief *is* its dashboard. |
| **Reviewer agent** | [reviewer-agent.md](interviews/reviewer-agent.md) | Prompted self-report against verify/review transcripts (§§ 5–7) | A spawned `reviewer` (`grant: ro`); verifies findings, reviews PRs, lives behind the human accept-gate. |

**Evidence weighting** (guide § 7, § 9, strongest → weakest): firsthand
past-behavior story > transcript coding / agent self-report > mock-up reaction.
The operator's daily stories outweigh a single adopter mock-up preference; every
ranking below carries the tier it rests on.

---

## 2. Personas

Four personas, one per segment — because dacli's users genuinely divide this way:
two humans who *watch* and two agents who *are watched*. The through-line that
organizes everything after is the split named in the guide's § 0: the dashboard
is a **read-only projection** of one JSON snapshot, while the sharpest needs are
**write** actions — and humans and agents pull that boundary in opposite
directions.

### P1 · "Priya" — the week-one adopter *(human, low tenure)*
Senior engineer, terminal-native, has driven one agent at a time but never a
swarm. Mental model coming in: *"an AI that does tickets."* Judges dacli by its
first hour. **Core need: legibility and reassurance** — teach me the tool, show
me the brakes, prove the cost is bounded. Her write-action interest is a
*discoverability* proxy ("I don't know the kill command"), not a mutate-need.
Churn moment: the **onboarding cliff** — bounces before the first agent ever runs
([adopter.md § 4.1 Q2](interviews/adopter.md#§41--first-contact)).

### P2 · "Sam" — the experienced operator *(human, high tenure)*
Runs the swarm most days; 3–6 agents at once; keeps `dacli agents --tail` in a
dedicated pane and reads it "like a heart monitor." Frames dacli as **a running
system to steer, not a backlog to manage** ([operator.md warm-up](interviews/operator.md#warm-up--context)).
**Core need: speed and scale** — glanceable instruments and a small set of
run-level write actions where observation and intervention happen in one glance.
This is the segment that puts *genuine* pressure on the write boundary — and the
one that, on reflection, argues its own steering wish back down to "a better
brief." Churn moment: the **scaling ceiling**, when the swarm outgrows `--tail`.

### P3 · The implementer agent *(agent, throughput-bound)*
A spawned coding agent driving a task to a PR. **Its dashboard is the brief**
([implementer-agent.md § 0](interviews/implementer-agent.md#0-the-one-thing-to-understand-first-the-brief-is-the-agents-dashboard)) —
it never renders the human UI. **Core need: not to be interrupted and not to be
blocked.** Every steering signal must arrive on a channel it already reads (its
context) at a turn boundary, or it is "theater the operator mistakes for a
correction." Biggest pain: a kill mid-edit that throws away salvageable,
uncommitted work.

### P4 · The reviewer agent *(agent, evidence-bound)*
A spawned reviewer (`grant: ro`); verifies claims and reviews PRs, **cannot check
its own boxes** — box-checking is owner-only
([reviewer-agent.md § 0](interviews/reviewer-agent.md#0-who-is-answering)). Already
lives behind the canonical human approve-gate. **Core need: reach a defensible
verdict without re-deriving the world, and have that verdict — with its trust
grade — land where the human already is.** The one segment that argues *for*
crossing the write boundary, and only for one action: working the accept-gate.

---

## 3. Human vs agent needs matrix

The acceptance criterion's core artifact: what each side needs, and where the two
diverge. The columns are the two humans (adopter, operator) and the two agents
(implementer, reviewer); the rows are the dimensions the interviews kept
returning to.

| Dimension | 🧑 Adopter (P1) | 🧑 Operator (P2) | 🤖 Implementer (P3) | 🤖 Reviewer (P4) |
|---|---|---|---|---|
| **Observability surface** | The **dashboard** — expects Datadog-style buttons | The **`--tail` pane + dashboard cockpit** | The **brief** (`dacli context`) — never sees the UI | The **brief + the artifact under review** (diff vs acceptance) |
| **What they need to *see*** | The **artifact**: "what did it change?" (diff / transcript link). Presence alone reads as blindness. | **Presence + liveness + direction**: is it advancing, *and* toward the right thing | **Its own budget-vs-band**, live sibling *claims*, its input artifacts named | **Diff annotated by acceptance box**, resolvable cited `file:line`, sibling verdicts + dissent |
| **Steering they want** | Buttons (a *discoverability* proxy — surfacing the command may satisfy it) | **Run-level** actions (kill, pause) *in the glance*; but suspects the real fix is a sharper brief | **Async constraint injection only** — synchronous chat is the top throughput risk | **None to itself** — steering a reviewer destroys the independence that makes a verdict worth gating |
| **Trust basis** | **UI-legible** — trusts what the dashboard *shows*; invisible safety = anxiety | **CLI-legible** — trusts the refusals it reads in its terminal | Trusts the **brief** — a rejected decision recorded pre-empts a wrong turn | Trusts **derived-from-log** verdicts; an in-place override breaks the invariant |
| **Cost / budget** | Live burn is the **#1 adoption gate** (surprise-bill fear) | Overspend is the **most expensive *silent* failure**; caught late or by proxy | **Blind to its own spend** mid-run; wants spent-vs-band in context | Over-band run = **review flag** (corners cut); consumes, doesn't own |
| **The write boundary** | *Wants* buttons because the CLI is undiscoverable | **"Operate-on-the-run OK; operate-on-the-work stays where the work is visible"** | *Wants the read-only projection kept* — it's the rail that stops synchronous interruption | Exactly **one** write action earns its place: working the accept-gate (H6) |
| **`--yolo` / autonomy** | A **cliff** — won't flip until cost, merges, brakes, refusals are watchable | A **considered choice** — has *seen* the governor and stop-file work | N/A — runs headless by contract | Wants a **tiered gate**: strong+reversible auto-applies under veto; weak/contested holds |
| **Top need** | **Onboarding to first-spawn**, then legible cost & provenance | **Burn-rate that yells (H7)** + **DAG view (H1)** | **Checkpoint grace on cancel** + budget-in-brief | **Trust grade + verdict tally loud, wherever a finding already surfaces** |

**The one-line read of the matrix:** humans need the system made **legible**
(adopter) and **glanceable + steerable at the run level** (operator); agents need
their **input channel enriched without being interrupted**. The two humans differ
by *tenure* (legibility vs speed); the two agents differ by *job* (throughput vs
evidence) but agree completely that the read-only projection is a **safety rail**,
not a limitation.

---

## 4. Key insights & tensions

Six findings recur across sources. Each is tagged with how many of the four
segments hit it and the evidence tier — the guide's ranking discipline
([§ 9](INTERVIEW_GUIDE.md#9-synthesis--evidence-discipline)).

### I1 · Budget visibility is the one need every segment shares *(4/4 — strongest recurrence)*
Live burn is the adopter's **#1 adoption gate** ([adopter.md Q7](interviews/adopter.md#§43--reaction-to-the-hypotheses-rq4)),
the operator's **most expensive *silent* failure** caught only late or by proxy
([operator.md Q13](interviews/operator.md#spend--history-rq1rq4)), a **review
flag** for the reviewer ([reviewer-agent.md § 4](interviews/reviewer-agent.md#4-reaction-to-each-hypothesis-rq4)),
and — reframed as *spent-vs-band in the brief* — the signal that would let the
implementer **self-govern before overspending** ([implementer-agent.md § 3.2](interviews/implementer-agent.md#32-budget--the-agent-is-blind-to-its-own-spend)).
Every number needed already exists (governor `windowSpent`, run actuals,
calibrated `role×model×runtime` bands). It is recorded and never shown as
rate-over-time. **This is the single safest, highest-value bet: universal demand,
zero new data, no boundary crossing.** The operator's sharpening matters — *"I
don't want to read a chart during a fire; I want the chart to yell"* — a
threshold that changes color at 1.5× band, not just a line.

### I2 · Steering vs throughput — the central tension, and it resolves against synchronous chat *(refuted by 3/4)*
The guide's headline tension. The pain is real: the operator wanted to type one
sentence to a wandering agent ([operator.md Q8](interviews/operator.md#steering-rq2));
the adopter wants buttons. But **synchronous chat-to-steer (naive H3) is refuted
by three independent segments**: the operator calls it a chat box that "rewards
bad briefs" and "scales to the six agents I can babysit and no further"
([operator.md Q8, reaction table](interviews/operator.md#solution-reaction--must--nice--no-rq4--7));
the implementer names it *"the single largest throughput risk in the entire
hypothesis set"* — it breaks the never-block headless contract, derails coherent
plans, and invites relitigation ([implementer-agent.md § 2](interviews/implementer-agent.md#2-when-human-chat--intervention-helps-vs-harms-mid-run-rq2--h3-the-core-question));
the reviewer says it *"destroys the independence that makes a verdict worth
anything"* ([reviewer-agent.md § 3](interviews/reviewer-agent.md#3-how-humans-should-approve--gate--steer-review-outcomes-from-the-ui-rq3--the-write-boundary)).
**Resolution: keep the *need*, refute the *feature*.** The correction the operator
wanted was information that belonged in the acceptance criteria; the safe delivery
is an **async constraint appended to the brief, consumed at a turn boundary** —
steering shaped like the thing that already works (a prior decision), just
delivered later.

### I3 · Escalation, not chat — pull beats push *(3/4 — meets the recurrence bar)*
The corollary to I2, and it arrives from three sides independently. The operator's
open floor: *"I'd rather the agent stop and ask me a sharp question than have me
lean over its shoulder"* — pull, not push ([operator.md open floor](interviews/operator.md#open-floor--un-hypothesized-needs-rq5--8)).
The implementer's missing channel is not the *ask* (that exists) but the
**reverse channel** — no resume-after-answer; when it asks, it STOPs, and the
answer only arrives as a *new* spawn's brief ([implementer-agent.md § 6.4](interviews/implementer-agent.md#6-answers-to-the-6-prompted-self-report-frame)).
The reviewer wants a **`dacli ask` that attaches to a task and *holds its gate*** so
a contested verdict pauses for a human rather than minting a false `confirmed`
([reviewer-agent.md § 7](interviews/reviewer-agent.md#7-un-hypothesized-need-rq5)).
Three segments, three angles, one primitive: **surface `dacli ask` live to the
human, and let the answer resume the blocked agent.** This is the un-hypothesized
need that, per the guide, *outranks* the § 2 hypotheses — and it is the steering
primitive to build *instead of* H3-chat.

### I4 · "Thinking vs. hung" — an observability gap the projections created *(2/4, operator past-behavior — strong)*
Every layer the operator has — `--tail`, `status`, Swarm — collapses to the
**same last line, re-skinned**, and none distinguishes an agent mid-`thinking`
from one wedged; resolving that ambiguity costs a four-tool context-switch several
times a day ([operator.md Q3, open floor](interviews/operator.md#observability-rq1)).
The adopter hits the same wall from the newcomer side: the snapshot carries
**presence, not artifact** — *who's running*, never *what they changed*
([adopter.md Q4](interviews/adopter.md#§42--observability-for-a-newcomer-rq1)). The
transcript *knows* (it has the `thinking` block); the projection throws it away.
The fix is one honestly-derived per-agent state — `thinking | acting | waiting |
stalled` — plus a bridge from the readout to the artifact (a "view transcript /
see the diff" link). **Both are read-only; this is the read-only half of H3.**

### I5 · The write boundary is real, and it splits cleanly: run vs work *(3/4 converge)*
The three segments that touch the boundary land on the *same* rule. The operator:
*"write actions that operate on the **run** (kill, pause) can live in the cockpit;
write actions that operate on the **work** (merge, edit content) must stay where
the work is visible"* — which maps onto [DESIGN.md § 2](https://github.com/mlnomadpy/dacli/blob/main/DESIGN.md)'s
*"runs agents, not work."* The implementer wants the read-only projection *kept*
as the rail against synchronous interruption. The reviewer allows exactly one
crossing — the accept-gate — because that gate *already exists* and is
code-enforced. **Verdict:** a confirmed pain does **not** auto-justify a dashboard
button. Pause and kill (operate-on-the-run) are the trustable crossings; merge and
steer (operate-on-the-work / -the-agent) are not.

### I6 · The adopter's whole trust bar clears without crossing the boundary *(1/4 — composite, code-grounded)*
The adopter's `--yolo` gate is four items — live cost + ceiling, a merge ledger, a
visible stop, a refusal feed ([adopter.md "before they'd trust --yolo"](interviews/adopter.md#what-the-dashboard-must-show-before-theyd-trust---yolo)) —
and **every one is a *read*** of state the system already produces. dacli's safety
is real and mostly on by default, but **CLI-legible and UI-invisible**; the same
safety that makes the operator confident makes the adopter anxious. The encouraging
finding: *newcomer demand does not pressure the read-only boundary at all.* The
pressure, where it's real, comes only from the operator.

---

## 5. Prioritized roadmap (RICE)

RICE = **(Reach × Impact × Confidence) ÷ Effort**, ranked high → low. Definitions,
tuned to this study:

- **Reach** — how many of the 4 segments the feature materially serves (1–4).
- **Impact** — 3 massive · 2 high · 1 medium · 0.5 low, for the segments it reaches.
- **Confidence** — evidence strength as a fraction, **capped at 0.8** because the
  whole study sits at the `unverified` trust floor (§ intro). 0.8 = recurs across
  ≥3 segments and/or firsthand past-behavior; 0.6 = two segments or one firsthand
  story; 0.4 = single-segment or mock-up reaction.
- **Effort** — rough engineering weeks. Read-only features that visualize
  *already-recorded* data are cheap; new write paths and honest state-derivation
  cost more.
- **MoSCoW** — the coarse bucket, for the roadmap-planning view.
- **Boundary** — Read (extends the projection) · Write · Brief (agent-facing) ·
  Onboarding.

Every row cites the interview(s) that motivate it. Score is rounded.

| Rank | Feature | Motivated by | R | I | C | E | **RICE** | MoSCoW | Boundary |
|---|---|---|:-:|:-:|:-:|:-:|:-:|:-:|:-:|
| 1 | **Burn-rate + ceiling, with a threshold that *yells*** (H7) | Operator MUST #1 ([op Q13](interviews/operator.md#spend--history-rq1rq4)); Adopter #1 ([ad Q7/Q9](interviews/adopter.md#§43--reaction-to-the-hypotheses-rq4)); Reviewer consumes; Implementer via brief-reframe | 4 | 3 | 0.8 | 2 | **4.8** | Must | Read |
| 2 | **`thinking \| acting \| waiting \| stalled` state + transcript/diff link** (H3-view + un-hyp.) | Operator top un-hyp. need ([op Q3, floor](interviews/operator.md#observability-rq1)); Adopter presence-vs-artifact ([ad Q4](interviews/adopter.md#§42--observability-for-a-newcomer-rq1)); Reviewer/Implementer view-OK | 3 | 2 | 0.8 | 2 | **2.4** | Must | Read |
| 3 | **Dependency / DAG view** (H1) | Operator MUST #2, *daily* manual reconstruction ([op Q6/Q15](interviews/operator.md#spend--history-rq1rq4)); critical path already computed | 2 | 3 | 0.8 | 1.5 | **3.2** | Must | Read |
| 4 | **Trust grade + verdict tally, loud, into the PR** (H6-view) | Reviewer #1 ([rv § 2, § 6](interviews/reviewer-agent.md#2-how-findings-should-surface-to-humans-rq1--rq3)); rides existing `--with-verdicts` | 2 | 2 | 0.8 | 1 | **3.2** | Must | Read |
| 5 | **Escalation surfaced live + resume-after-answer** (`dacli ask` in cockpit) | Operator floor ([op floor](interviews/operator.md#open-floor--un-hypothesized-needs-rq5--8)); Implementer reverse-channel ([im § 6.4](interviews/implementer-agent.md#6-answers-to-the-6-prompted-self-report-frame)); Reviewer contested-gate ([rv § 7](interviews/reviewer-agent.md#7-un-hypothesized-need-rq5)) | 3 | 2 | 0.8 | 3 | **1.6** | Should | Write* |
| 6 | **Pause / resume the *governor*** (H5, scheduler only) | Operator MUST #3, real recent ([op Q10](interviews/operator.md#steering-rq2)); Implementer OK *iff* governor-only ([im § 4](interviews/implementer-agent.md#4-reaction-to-each-hypothesis-from-the-agents-side-rq4)) | 2 | 2 | 0.8 | 1.5 | **2.1** | Should | Write |
| 7 | **Provenance bundle: merge ledger + visible stop + refusal feed** | Adopter skeptic gate #3 ([ad Q9](interviews/adopter.md#§44--adoption--trust-rq3-rq4)); Operator merge-queue view ([op Q11](interviews/operator.md#steering-rq2)) | 2 | 3 | 0.6 | 3 | **1.8** | Should | Read |
| 8 | **Onboarding that reaches the first spawn** (not a dashboard feature) | Adopter #1, gates all adoption ([ad Q2/Q10](interviews/adopter.md#top-3-features-for-the-new-adopter-segment)); `wscore.go:87-106` dead-ends at `overview` | 1 | 3 | 0.8 | 1 | **2.4** | Must | Onboard |
| 9 | **Checkpoint grace on cancel** (SIGTERM → *commit* → SIGKILL) + kill-from-UI (H2) | Implementer #1, highest agent pain ([im § 3.4](interviews/implementer-agent.md#34-being-about-to-be-cancelled--the-biggest-single-gap-on-the-kill-path)); Operator NICE kill ([op Q7](interviews/operator.md#steering-rq2)) | 2 | 2 | 0.7 | 2 | **1.4** | Should | Write/Brief |
| 10 | **Budget-vs-band line in the brief** (H7, agent-facing reframe) | Implementer #2 ([im § 3.2](interviews/implementer-agent.md#32-budget--the-agent-is-blind-to-its-own-spend)); data already exists | 1 | 2 | 0.7 | 1.5 | **0.9** | Should | Brief |
| 11 | **Diff annotated by acceptance box; unbacked boxes flagged** | Reviewer #2, the defect it most-often misses ([rv § 1](interviews/reviewer-agent.md#1-what-i-need-to-review-well-rq1--observability-reviewers-cut)) | 1 | 2 | 0.6 | 2 | **0.6** | Could | Read |
| 12 | **Retro timeline** (H8) | Operator NICE, retro not live ([op Q14](interviews/operator.md#spend--history-rq1rq4)); Adopter learning aid ([ad Q7](interviews/adopter.md#§43--reaction-to-the-hypotheses-rq4)); Reviewer adjudication ([rv § 4](interviews/reviewer-agent.md#4-reaction-to-each-hypothesis-rq4)) | 3 | 1 | 0.7 | 3 | **0.7** | Could | Read |
| 13 | **Conditional approve-gate from the UI** (H6, evidence-tiered) | Reviewer top write ([rv § 5](interviews/reviewer-agent.md#5-where-the-approve-gate-adds-value-vs-slows-the-swarm)); Adopter skeptic ([ad Q9](interviews/adopter.md#§44--adoption--trust-rq3-rq4)); Operator *queue only*, not one-click approve ([op Q11](interviews/operator.md#steering-rq2)) | 2 | 2 | 0.6 | 3 | **0.8** | Could | Write |

\* Item 5's write is *agent-facing plumbing* (resume channel), not a
dashboard-mutates-work button — it delivers on I3, the pull-not-push primitive,
and is the sanctioned replacement for H3-chat.

### Won't build (refuted or out-of-doctrine)

| Feature | Why refuted | Evidence |
|---|---|---|
| **H3-steer as a synchronous chat box** | Top throughput risk; rewards bad briefs; destroys reviewer independence; can't reach a headless agent anyway | Refuted by 3/4 — [op](interviews/operator.md#solution-reaction--must--nice--no-rq4--7), [im § 2/§ 5.1](interviews/implementer-agent.md#5-features-that-would-harm-agent-throughput-if-added-naively-acceptance-flagged), [rv § 3](interviews/reviewer-agent.md#3-how-humans-should-approve--gate--steer-review-outcomes-from-the-ui-rq3--the-write-boundary) |
| **H4 — drag to re-prioritize** | "Never once blocked me"; irrelevant at a newcomer's four-task backlog; neutral to both agents | Operator NO ([op Q9](interviews/operator.md#steering-rq2)); adopter "pure operator" |
| **One-click PR *approve* without the diff** | A footgun — invites approving on vibes; approval belongs where the evidence is | Operator ([op Q11/Q12](interviews/operator.md#cost--trust-rq3)) |
| **Kill wired straight to SIGKILL** (no commit grace) | Destroys salvageable uncommitted work every time | Implementer ([im § 3.4/§ 5.2](interviews/implementer-agent.md#34-being-about-to-be-cancelled--the-biggest-single-gap-on-the-kill-path)) |
| **Pause that suspends a *running agent*** (vs the governor) | Stale locks, half-written files, spurious stall-timeouts | Implementer ([im § 5.3](interviews/implementer-agent.md#5-features-that-would-harm-agent-throughput-if-added-naively-acceptance-flagged)) |
| **A mandatory human gate on `pr --auto`** | Converts autonomous claim→PR→ship into a bottleneck — the cost the swarm exists to avoid | Implementer ([im § 5.4](interviews/implementer-agent.md#5-features-that-would-harm-agent-throughput-if-added-naively-acceptance-flagged)); Reviewer tiered-gate ([rv § 5](interviews/reviewer-agent.md#5-where-the-approve-gate-adds-value-vs-slows-the-swarm)) |

---

## 6. Ready to build — the shortlist

The subset that is **high-value, read-only, and built on data the system already
records** — the guide's safest graduations, clearing no boundary and needing no
new collection machinery. In priority order:

1. **Burn-rate + ceiling with a yelling threshold (RICE 4.8, item 1).**
   Universal demand (4/4), highest score, zero new data. The one feature every
   segment named. Ship the color-changing 1.5×-band alert, not just a chart.

2. **`thinking | acting | waiting | stalled` state + a transcript/diff link
   (RICE 2.4, item 2).** Kills the operator's daily thinking-vs-hung
   context-switch and the adopter's presence-vs-artifact blindness with one
   honestly-derived field the transcript already contains.

3. **Dependency / DAG view (RICE 3.2, item 3).** The operator's single most
   repeated manual reconstruction; `internal/spm/criticalpath.go` already computes
   the chain — this only *draws* it.

4. **Trust grade + verdict tally rendered into the PR (RICE 3.2, item 4).**
   Extends the existing `dacli pr --with-verdicts` path; makes
   `refuted < unverified < confirmed` a loud, first-class column wherever a
   finding already appears. No new screen.

5. **Onboarding to first-spawn (RICE 2.4, item 8).** Not a dashboard feature, but
   the cheapest change with the largest first-hour effect and the one that gates
   whether an adopter ever reaches items 1–4. `init`'s getting-started list must
   not dead-end at `overview`.

**Everything on this shortlist is a `Read` or `Onboard` boundary** — none touches
the write doctrine, all five stand on data dacli already produces, and four of the
five carry ≥0.8 confidence. This is where the next quarter should start.

The **operator's own forced trade-off** independently corroborates the top of this
list: asked for only two features next quarter, they picked **H7 (burn-rate) and
H1 (DAG)** and gave up *everything write-side without much pain*
([op § 7 trade-off](interviews/operator.md#solution-reaction--must--nice--no-rq4--7)).

---

## 7. Write-boundary verdicts (RQ3)

For every write hypothesis, the guide demands an explicit verdict: does the
evidence justify crossing the read-only projection, or is the pain better served
in the CLI / by a better brief? A confirmed pain is necessary but **not
sufficient**.

| Hypothesis | Verdict | Rationale |
|---|---|---|
| **H2 — kill from UI** | **Cross, conditionally.** | Only *next to the signal that triggers it* and only as SIGTERM → agent-commit-grace → SIGKILL. Operate-on-the-run; blast radius is one already-watched run. ([op Q7/Q12](interviews/operator.md#cost--trust-rq3), [im § 3.4](interviews/implementer-agent.md#34-being-about-to-be-cancelled--the-biggest-single-gap-on-the-kill-path)) |
| **H3-steer** | **Do not cross.** | Refuted by 3 segments. Replace with async constraint-injection + the escalation channel (I2, I3). |
| **H4 — drag-reprioritize** | **Do not build.** | No segment is blocked by its absence. |
| **H5 — pause/resume** | **Cross — the safest write.** | *Only* the governor (stop new spawns), never a running agent. Operate-on-the-scheduler; touches no agent's work. ([op Q10](interviews/operator.md#steering-rq2), [im § 4](interviews/implementer-agent.md#4-reaction-to-each-hypothesis-from-the-agents-side-rq4)) |
| **H6 — approve-gate** | **Cross, but evidence-tiered.** | Strong + reversible panels **auto-apply under human veto** (`pr --auto`); **only weak / contested / irreversible** cases surface an accept/reject button — *with the diff*. A gate that fires on every clean panel is the bottleneck the guide warned about. Never a one-click approve without the code. ([rv § 5](interviews/reviewer-agent.md#5-where-the-approve-gate-adds-value-vs-slows-the-swarm), [op Q11](interviews/operator.md#steering-rq2)) |

**The durable rule (I5):** *operate-on-the-run* actions (kill, pause) may enter
the cockpit; *operate-on-the-work* actions (merge, edit) stay where the work is
visible. This is not merely a preference — it maps onto
[DESIGN.md § 2](https://github.com/mlnomadpy/dacli/blob/main/DESIGN.md)'s "runs
agents, not work," which is why it reads as a durable boundary rather than a mood.

---

## 8. Where this goes next (evidence discipline)

Per [§ 9](INTERVIEW_GUIDE.md#9-synthesis--evidence-discipline), nothing here is
`confirmed`: the study is one firsthand operator transcript plus three composite /
self-report sources, all at trust-floor `unverified`. To graduate any roadmap item
to a [PROPOSALS.md](../PROPOSALS.md) entry with acceptance tests:

- **Run the live scripts against ≥3 real users per human segment** — at least one
  operator beyond the one transcript here, and at least one adopter who *bounced* —
  transcripts in `docs/research/transcripts/`. Code the agent segments across a
  **stratified sample of real run transcripts** (successes, refusals, timeouts,
  killed runs, bounced PRs), not one self-report each.
- **A need graduates when it recurs across ≥3 independent real sources**, each with
  a cited quote or turn-ref. The strongest leads to test first — because they
  already recur across the composites — are **I1 (burn visibility, 4/4)** and
  **I3 (escalation-not-chat, 3/4)**.
- **A need that contradicts a non-goal** goes to the design audit
  ([REVIEW.md](../REVIEW.md)), not around it. None of the `Read`-boundary shortlist
  does; the write items (H2, H5, H6) press on
  [DESIGN.md § 2](https://github.com/mlnomadpy/dacli/blob/main/DESIGN.md) and must
  clear that gate explicitly, per the verdicts in § 7.

The encouraging shape of the whole study: **the highest-value, most-recurring needs
are read-only** (I1, I4, I6, and four of the five shortlist items). The dashboard
can get materially better for every segment before the write boundary is ever
tested — and when it is tested, the evidence says test *pause* and the *tiered
accept-gate* first, and never ship *steer-as-chat* at all.

---

_This synthesis is versioned with the code and the guide. When live interviews
replace the composite sources, re-run this rollup — a roadmap ranked on stale
transcripts is incomparable data. Amend via PR._
