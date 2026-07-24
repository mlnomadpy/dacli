# Interview — Human adopter: trust & onboarding needs

Status: **research artifact — a code-grounded composite, not a real transcript.**
This document answers the [Human adopter script (§4)](../INTERVIEW_GUIDE.md#4-script--human-adopter)
of the discovery guide from the point of view of the **new-adopter** segment:
someone who has run `dacli init` once (or is deciding whether to) and judges the
tool by its first hour.

**Read this as a lead, not a fact.** Per the guide's evidence discipline
([§9](../INTERVIEW_GUIDE.md#9-synthesis--evidence-discipline)), a single source is
`unverified` — a need is `confirmed` only when it recurs across ≥3 independent
sources. No real adopter has been interviewed yet (`docs/research/transcripts/`
is empty), so this is a **composite persona**, not a recording of a person. What
keeps it honest rather than fiction: every reaction below is anchored to *actual
dacli behavior*, cited `file:line` the way an agent finding cites its evidence —
the first-run output a newcomer really sees, the read-only dashboard they really
hit, the `--yolo` checkpoint they really give up. It is a hypothesis-shaped
instrument that stands in until live interviews replace it; when they do, this
file's claims graduate or get refuted, they don't get trusted as-is.

**Trust floor: unverified.**

---

## The persona — "Priya," week-one adopter

A senior backend engineer at a 12-person startup. Ships Go and TypeScript, lives
in the terminal, has used `claude` and `codex` CLIs one-agent-at-a-time but never
a *swarm*. A teammate dropped a link; she ran `brew install`, `dacli init` in a
scratch repo, and poked at it for an evening. She has **not** built scars yet —
she has expectations, and the gap between them and reality is the whole signal.
Where a real interview would carry a name tag and a git-ignored transcript, this
carries `file:line`.

Her mental model coming in: *"an AI that does tickets."* The reality — a
governed process loop that spawns child CLIs, mints them identities, and merges
their PRs — is bigger than that model, and the friction is where the two collide.

---

## §4.1 — First contact

> **Q1. What made you try dacli? What did you expect it to do before you ran anything?**

> "A teammate said 'it runs a *team* of Claudes, not one.' That's what sold me —
> I'm already babysitting one agent in a terminal, and the promise was I'd stop
> babysitting. Before I ran anything I pictured a dashboard: a board of tickets,
> little agent avatars crawling across them, a big green 'go' button. Honestly I
> pictured something closer to a CI dashboard than a CLI. The README's hero even
> shows a loop diagram — `context → spawn → wait → accept → ship` — so I thought
> the dashboard *was* the product and the CLI was the plumbing."

*Coding note (RQ4/RQ5):* the expectation is **UI-first**. The product is
CLI-first — the dashboard is explicitly a **read-only projection**
(`internal/features/dashboard/ui/DESIGN.md` §0; `internal/features/dashboard/dashboard.go:66`).
That inversion is the root of most of what follows.

> **Q2. Walk me through your first session. Where did you get stuck or confused?**
> *(Do not rescue — let her narrate.)*

> "`dacli init` worked instantly — it printed `initialized workspace` and then a
> nice little **Getting started** list: `whoami`, `project add`, `task add`,
> `next`, `overview`. So I did exactly that. Added a project, added a task with
> `--accept`, ran `next`. Fine. It felt like a task tracker.
>
> Then I got *stuck* — because the getting-started list ends at `overview`.
> Nothing on that list told me how to make an agent actually *do* the task. I
> went looking. I found `spawn` in `dacli help`, but I had to guess the flags:
> `--role`? `--grant`? What's a *grant*? What's a *role*? The init flow walked me
> from empty workspace to a briefed task and then… stopped, right at the moment
> the tool gets interesting. I never even learned the dashboard existed until I
> grepped the docs."

*Coding note (RQ1/RQ5 — the strongest first-run finding):* the onboarding path
**dead-ends before the fleet**. `printGettingStarted`
(`internal/features/wscore/wscore.go:87-106`) lists exactly five steps —
`whoami`, `project add`, `task add`, `next`, `overview` — and names neither
`spawn` nor `dacli dashboard` (`internal/features/dashboard/dashboard.go:31`).
The newcomer's first-hour mental model is therefore *"a task tracker,"* not *"an
agent swarm,"* because the tool never points them at the swarm. This is the exact
expectation gap §4 exists to probe. (Filed as a finding on task 139.)

> **Q3. The first time you opened the dashboard, what did you expect to be able
> to do? What did you try to click that didn't do anything?**
> *(Directly probes the read-only-projection surprise, RQ3.)*

> "Once I found `dacli dashboard` it opened a slick dark 'mission control' page —
> projects, a burndown, a live agent list. Genuinely good-looking. So I treated
> it like an app. I tried to **click a task to open it** — nothing. I looked for
> a **'+ new task'** button — none. When an agent showed up in the swarm list I
> tried to **click it to see its transcript** — nothing. I right-clicked. I
> looked for a kebab menu. There's no affordance because there's no *action* —
> it's a poll of a JSON snapshot every couple seconds and nothing on it mutates
> anything. It took me a full minute of clicking dead pixels to realize it's a
> *readout*, not a *console*. It looks like Datadog, so I expected Datadog's
> buttons."

*Coding note (RQ3 — the read-only surprise, confirmed against code):* the page
polls `GET /api/state` and *"nothing in the UI ever mutates the workspace"*
(`internal/features/dashboard/ui/DESIGN.md` §0). The visual language is a
*"dark operations console… instruments, not a marketing page"* (§1) — which is
exactly why a newcomer reads it as *interactive*. **The more mission-control it
looks, the more the read-only-ness violates expectation.** This is a newcomer
tax the operator never pays, because the operator already knows the CLI is the
control surface.

---

## §4.2 — Observability for a newcomer (RQ1)

> **Q4. When you had agents running, did you feel like you knew what was going
> on? What made you nervous?**

> "No — and the nervousness was specifically *trust*, not *information*. I had
> the swarm list showing me `a-nhkth9j71n · designer · claude · 6m · 40s ago`.
> Great, but I don't have the vocabulary yet. What's a *designer* role doing to
> my code? Is `40s ago` good or hung? For the operator that last-activity
> timestamp is a liveness tell; for *me* it was noise, because I had no baseline
> for what 'normal' looks like. What made me nervous wasn't a missing panel — it
> was that agents were **editing my repo and I couldn't see the edits**, only
> that *an* agent existed. I wanted a diff. I wanted 'here's what it changed so
> far.' The dashboard shows me *who's running*; I wanted *what they're doing to
> my files.*"

*Coding note (RQ1):* the snapshot carries **presence, not artifact** — `run_id`,
`child`, `task`, `role`, `runtime`, `pid`, `runtime_secs`, `last_activity`
(`internal/features/dashboard/ui/DESIGN.md` §0 data contract). There is no diff,
no per-agent transcript, no "files touched." For a *seasoned operator* that's
correct — they drop to `dacli logs -f` or read the transcript file. For a
*newcomer* the absence reads as blindness, because they don't yet know the
transcript file exists. **The newcomer need is not a new panel; it's a bridge
from the readout to the artifact** (a "view transcript" / "see the diff" link),
which happens to be the read-only *view* half of H3.

> **Q5. Which panel made immediate sense? Which one did you not understand?
> (Overview / Board / Burndown / Swarm.)**

> - **Overview — immediate.** "Projects, a stage, counts. I got it in a glance.
>   This is the one panel that spoke my language on day one."
> - **Board — mostly.** "open / active / blocked / done columns. Fine. Though
>   'blocked' scared me — blocked by *what*? It's a count, not a reason."
> - **Burndown — no.** "Points? `done_points` vs `remaining_points`,
>   `unestimated: 4`. I never estimated anything, so my burndown was mostly
>   'unestimated' and the bar looked broken. As a newcomer I have no points yet,
>   so the one chart that's supposed to show momentum showed me *nothing moving*
>   and I read that as 'the tool isn't working.' An empty burndown is scarier to
>   a newcomer than to an operator who knows it fills in."
> - **Swarm — visually yes, semantically no.** "I saw the agents. I didn't know
>   what any of the *roles* meant, so the most important panel was the most
>   opaque."

*Coding note (RQ1, newcomer-specific):* the burndown's empty/unestimated state
(`unestimated` surfaced as a caption, bar drawn from points only —
`internal/features/dashboard/ui/DESIGN.md` §0) is **information-correct but
confidence-destroying** for someone with zero estimated tasks. The operator reads
an empty burndown as "early"; the newcomer reads it as "broken." Same pixels,
opposite trust signal.

---

## §4.3 — Reaction to the hypotheses (RQ4)

*Kept deliberately lighter than the operator script — the adopter reacts to
possibilities, which is the weakest evidence tier ([§7](../INTERVIEW_GUIDE.md#7-solution-reaction-last-after-the-problem-is-fully-heard)).
Weighted accordingly.*

> **Q6. When you imagined controlling this thing, what did you picture — a
> dashboard you click, or commands you type? Why?**

> "A dashboard I click — 100%, and I think that's *because* I'm new. I don't have
> the commands memorized, so a button is discoverable and a flag is a thing I
> have to *know exists*. The operator has the muscle memory; a terminal is faster
> for them. For me the terminal is a wall of `--role --grant --claim --detach`
> that I have to learn before I can do anything. So my instinct is 'give me the
> buttons' — but I'll admit that instinct is about *my ignorance*, not about
> what's *safe*. I'd probably want the buttons for reading and the terminal for
> anything scary."

*Coding note (RQ3):* this is the crux the guide names — the newcomer *wants* the
write actions in the UI precisely because the CLI is undiscoverable to them,
while [DESIGN.md §2](../../DESIGN.md)'s *"runs agents, not work"* boundary and the
read-only doctrine push the other way. The newcomer's preference is a
**discoverability** signal, not a **need-to-mutate-from-UI** signal — the fix
might be surfacing the *command* ("kill this → `dacli kill a-nhkth9j71n`,
click to copy") rather than a live button.

> **Q7. Of these, which would have made your first week easier: token burn as a
> chart (H7), a timeline of what happened (H8), a picture of what's blocked (H1)?
> Rank them for your stage.**

> "For *my* stage:
> 1. **Token burn (H7) — first, and it's not close.** My #1 fear adopting an
>    autonomous tool is a surprise bill. I want to watch the money move in real
>    time. Right now I find out cost *after*. A live burn line is what would let
>    me sleep while it runs.
> 2. **Timeline (H8) — second.** After a run I have no idea what happened. A
>    scrubbable 'here's the story of the last hour' would teach me the tool as
>    much as inform me. For a newcomer the timeline is a *learning* aid, not just
>    an audit trail.
> 3. **Blocked-graph (H1) — a distant third.** I have four tasks. I don't have a
>    dependency problem yet. Ask me in month three. This is an *operator* feature
>    — it only matters at a backlog size I don't have."

*Coding note (RQ4):* the newcomer ranking **inverts** the operator's likely one.
H1 (DAG view) is top-tier for an operator drowning in `blocked/`
(`internal/spm/criticalpath.go` computes the critical path already), but bottom
for a newcomer with a four-task backlog. H7/H8 — both **read-only**, both cheap
to reconcile with the projection — are where newcomer and operator value
*overlap*, which makes them the safest bets. Spend/history is *"Operator,
Adopter"* on the hypothesis table (H7, H8 in [§2](../INTERVIEW_GUIDE.md#2-the-feature-hypotheses-to-probe)),
and this interview confirms the adopter half.

---

## §4.4 — Adoption & trust (RQ3, RQ4)

> **Q8. What almost made you stop using it? What would have made you stop?**

> "What *almost* stopped me: the onboarding cliff at Q2 — I nearly bounced when
> the getting-started list ran out and I couldn't figure out how to make an agent
> run. Ten more minutes of 'what's a grant' and I'd have closed the tab.
>
> What *would* have stopped me cold: if an agent had merged something to `main`
> and I found out after. It didn't — the default loop pauses between cycles and
> asks me to re-run — but I didn't *know* that when I started, and the fear alone
> is a churn risk. 'Autonomous' and 'merges my code' in the same sentence is
> terrifying until someone shows you the brakes."

> **Q9. If you're deciding whether your team adopts this, what does the dashboard
> need to show a *skeptic* looking over your shoulder?**

> "My CTO is the skeptic, and she'll ask three things, in order:
> 1. **'What did it change, and did a human approve it?'** — show me the PRs it
>    opened and who merged them. If the answer is 'it merged itself,' show me the
>    gate that a human passed.
> 2. **'What is this costing?'** — a number, live, per run. (That's H7 again.)
> 3. **'How do I stop it *right now*?'** — one obvious, always-visible stop. Not
>    a flag I have to remember; a big red thing.
> If the dashboard answers those three, she signs off. If it only shows
> burndown and agent uptime, she says 'cute' and we don't adopt. The dashboard
> today is built for someone who already trusts it — the skeptic needs the
> *provenance and the brakes*, not the *instruments*."

*Coding note (RQ3, adoption-gating):* none of the skeptic's three are on the
snapshot today (`internal/features/dashboard/ui/DESIGN.md` §0 carries projects,
burndown, agents — no PR provenance, no live cost, no stop control). The brakes
*exist in the CLI* — `touch .dacli/STOP` halts at the next checkpoint, the
between-cycle pause is the default human touchpoint
(`internal/features/orchestration/orchestration.go:358-362`), and merges route
through PRs (`dacli pr --auto`) — but they are **invisible on the surface a
skeptic is shown.** Adoption is gated less by capability than by *legible*
capability.

> **Q10. (If they bounced) What were you doing the moment you decided to stop?
> What would have had to be different?**

> "I didn't fully bounce, but the moment I *considered* it: staring at
> `dacli help` output trying to reverse-engineer `spawn`, twenty minutes in, with
> zero agents ever having run. If `init` had ended with *'now run
> `dacli spawn --task 001` to put an agent on it'* — one line — I'd have been in
> the swarm in the first five minutes instead of the fortieth. The thing that
> would've had to be different is tiny: **the onboarding needs to reach the
> first spawn, not stop at the first task.**"

---

## The two questions the acceptance criteria name directly

### Trust concerns about autonomous agents merging code

The adopter's fear concentrates on one sentence: *an agent merges to `main` and
I find out after.* Grounded against the code, here is what actually protects them
and — critically — **what they can't see protecting them:**

| Fear | The real brake | Is it *visible to a newcomer?* |
|---|---|---|
| "It'll merge without me." | Default loop **pauses between cycles** for the operator to re-run; unbounded runs are **refused** unless you opt in (`orchestration.go:136-137`, `:358-362`). | ❌ Not surfaced anywhere in the UI or init. |
| "It'll merge broken code." | Merges go through **PRs** (`dacli pr --auto` queues native auto-merge behind required checks); `accept` verifies acceptance before closing. | ❌ PRs/gates absent from the dashboard snapshot. |
| "A poisoned brief will hijack an agent." | `taint` **refuses to spawn** onto a brief in an injected source's blast radius (README "trust & taint gates"). | ❌ Taint state not on the newcomer's surface. |
| "Agents will clobber each other." | A `--claim` path conflict is **refused** before it can clobber a sibling. | ❌ Claims invisible in UI. |

**The newcomer-specific finding:** dacli's safety is *real and mostly on by
default*, but it is **CLI-legible and UI-invisible**. The operator trusts it
because they've read the refusals in their terminal. The adopter can't, so the
same safety produces *anxiety* instead of *confidence*. Trust for this segment is
not a capability problem — it's a **surfacing** problem.

### What the dashboard must show before they'd trust `--yolo`

`--yolo` is the specific act of trust the guide's write-boundary question circles.
In the code it means *"no between-cycle checkpoint pause"*
(`orchestration.go:75`) — the one standing human touchpoint is removed, and
combined with `dacli pr --auto` the loop runs `implement → PR → auto-merge →
regenerate backlog` indefinitely with **no human in the loop**
(`orchestration.go:358-362`; README §"The perpetual loop"). An adopter will not
flip that switch on faith. Before they would, the dashboard must show — and none
of these are on the snapshot today:

1. **Live cost, with a ceiling and distance-to-it.** "Spent $X of your $Y window,
   burning $Z/hr, ~N hours left." The governor already enforces a token window
   (`--window-tokens`); the *number* must be visible, not just enforced. (H7,
   read-only, **the single strongest adopter ask.**)
2. **A merge ledger.** Every PR the loop opened and merged, with its checks —
   "12 merged green, 0 forced, 1 waiting." Autonomy is only trustable if its
   output is auditable *as it happens*, not reconstructable after.
3. **The brakes, always on screen.** A visible stop affordance and the fact that
   `touch .dacli/STOP` exists — "this run halts at the next checkpoint if you hit
   this." The kill switch being *invisible* is why `--yolo` feels like a cliff.
4. **A refusal feed.** Show the gates *working*: "refused 1 spawn (tainted),
   refused 1 claim conflict." A newcomer trusts an autonomous system far more
   when they can watch it *say no* to itself than when it only ever says yes.

The through-line: **an adopter trusts `--yolo` in proportion to how legibly the
dashboard shows the system restraining itself.** Cost ceiling, merge provenance,
a visible stop, and refusals-in-action — the guarded autonomy, made watchable.
Whether these become dashboard *reads* (safe, they extend the projection) versus
*write* controls (the boundary [RQ3](../INTERVIEW_GUIDE.md#1-research-questions)
guards) is the synthesis question — but every item above is a **read**, which is
the encouraging part: the adopter's trust bar can be cleared without crossing the
read-only boundary at all.

---

## Reaction to every hypothesis, from the adopter's chair

Ranked by *this segment's* evidence, weakest tier (mock-up reaction) noted per
[§7](../INTERVIEW_GUIDE.md#7-solution-reaction-last-after-the-problem-is-fully-heard).

| # | Hypothesis | Adopter reaction | Newcomer relevance | Write? |
|---|---|---|---|---|
| **H7** | Budget / token burn charts | **"My #1. The surprise-bill fear.** Show me the money live." | **High** — the top adoption gate. | Read ✅ |
| **H8** | Retro timeline | "Second. It **teaches me the tool** as much as audits it." | **High** — a learning aid for someone with no scars. | Read ✅ |
| **H3-view** | Live transcript *view* (read half) | "Yes — a **'see what it changed'** link. I want the artifact, not just the agent's name." | **High** — closes the presence-vs-artifact gap (Q4). | Read ✅ |
| **H6** | Approve-gate a PR from the UI | "This is the **skeptic's** feature (Q9). A visible human-approved gate is what gets my CTO to sign off." | **Med-High** — matters for *team* adoption, not solo first hour. | Write ⚠ |
| **H2** | Cancel / kill from the UI | "I want it because I **don't know the kill command** — but that's my ignorance. Surfacing the command might serve me better than a button." | **Med** — discoverability, not a mutate-need. | Write ⚠ |
| **H5** | Pause / resume the loop | "The **brake I need for `--yolo`.** But I'd accept it as a visible `.dacli/STOP` I can see, not necessarily a UI button." | **Med** — trust-blanket for autonomy. | Write ⚠ |
| **H1** | Task dependency / DAG view | "**Ask me in month three.** I have four tasks; I have no dependency problem yet." | **Low** — an operator feature at a backlog size I don't have. | Read ✅ |
| **H4** | Drag to re-prioritize | "Irrelevant at four tasks. **Pure operator.**" | **Low** — no backlog to reorder. | Write ⚠ |

The shape of the table is the finding: the adopter's high-value set is **almost
entirely read-only** (H7, H8, H3-view), and their write-action interest (H2, H5)
is a *discoverability* proxy that a surfaced-command might satisfy more cheaply
than a button. The write hypotheses that press hardest on the boundary (H3-steer,
H4) are the *least* relevant to this segment. **Newcomer demand does not
pressure the read-only boundary** — that pressure, if it's real, will come from
the operator segment (§3), not this one.

---

## Top 3 features for the new-adopter segment

Ranked by first-hour/first-week impact for someone judging the tool by its first
hour. The onboarding fix is #1 because **no feature helps a user who bounces
before reaching it.**

1. **Onboarding that reaches the first spawn, not the first task.**
   The single highest-leverage change and the cheapest. `init`'s getting-started
   list must not dead-end at `overview`
   (`internal/features/wscore/wscore.go:87-106`); it must carry the newcomer to a
   *running agent* and to `dacli dashboard`. Nearly the whole first-hour
   confusion (Q2, Q8, Q10) traces here. Not a dashboard feature at all — an
   onboarding one — which is exactly why it's easy to miss and highest-impact.

2. **Live cost, ceiling, and burn rate on the dashboard (H7).**
   The top adoption *and* trust ask: the surprise-bill fear (Q7) and the
   `--yolo` gate (Q9 #2) are the same feature. Read-only, so it extends the
   projection without touching the boundary. The one thing that lets an adopter
   walk away from a running loop.

3. **Provenance + brakes, made visible: a merge ledger, a visible stop, and a
   refusal feed.**
   The bundle that answers the skeptic's three questions (Q9) and clears the
   `--yolo` trust bar: *what did it merge, who approved, how do I stop it, what
   did it refuse.* Every piece is a **read** of state the system already produces
   (PRs, `.dacli/STOP`, taint/claim refusals) — the capability exists; only its
   *legibility* is missing. Turns dacli's real, default-on safety from a source
   of anxiety into a source of confidence.

---

## Needs unique to the newcomer vs. the experienced operator

The acceptance criterion the guide leans on hardest: telling apart what a
*newcomer* needs from what an *operator* needs, so a confirmed newcomer pain
isn't mistaken for a universal one.

| Dimension | **Newcomer (adopter)** | **Experienced operator** |
|---|---|---|
| **Control surface** | Wants **buttons** — the CLI is undiscoverable, a flag is a thing you must *know exists*. | Wants the **terminal** — muscle memory makes flags faster than a mouse. |
| **Why they want write-in-UI** | **Discoverability** ("I don't know the kill command"). Might be met by *surfacing the command*, not a live button. | **Speed / ergonomics** at scale — a genuine mutate-from-glance need. |
| **The dashboard's job** | **Teach and reassure** — legible provenance, a learning timeline, visible brakes. | **Glanceable instruments** — liveness, momentum, spend, at a swarm size where reading transcripts doesn't scale. |
| **Observability need** | The **artifact** — "what did it change?" (a diff, a transcript link). Presence alone reads as blindness. | **Presence + liveness** — `last_activity`, RAM/CPU, is-it-hung. They already know how to reach the artifact. |
| **Burndown / DAG (H1, H8)** | Burndown is **confusing** (no points yet, empty bar reads as "broken"). DAG is **premature** (four tasks). | Burndown is **momentum**; DAG (H1) is **critical-path relief** at real backlog size. |
| **Trust basis** | **UI-legible** — trusts what the dashboard *shows*. Can't read CLI refusals, so invisible safety = anxiety. | **CLI-legible** — trusts the refusals they read in their terminal; the safety is already visible to them. |
| **`--yolo`** | A **cliff** — won't flip it until cost, merges, brakes, and refusals are watchable. | A **considered choice** — flips it because they've *seen* the governor, thrash guard, and stop file work. |
| **Churn moment** | The **onboarding cliff** (Q2/Q10) — bounces before the first agent runs. | The **scaling ceiling** — bounces when the swarm outgrows `agents --tail`. |
| **Top feature** | **Onboarding to first-spawn**, then legible cost & provenance. | **DAG view (H1)** and **steer (H3)** at scale. |

**The one-line synthesis (unverified — a lead for the ≥3-source bar):** the
newcomer's needs are about **legibility and reassurance** — teach me the tool,
show me the brakes, prove the cost is bounded — and are met almost entirely by
**read-only** surfacing of capability that already exists. The operator's needs
are about **speed and scale** and are where the genuine pressure on the
read-only *write* boundary lives. Confusing the two would build operator write-
controls to solve a newcomer *discoverability* problem — an expensive answer to
the wrong question. The cheapest, highest-impact adopter wins (onboarding to
first-spawn; legible cost, provenance, and brakes) **cross no boundary at all.**

---

## Where this goes next (evidence discipline)

Per [§9](../INTERVIEW_GUIDE.md#9-synthesis--evidence-discipline), nothing here is
`confirmed` — it is one composite source at trust-floor `unverified`. To graduate
any of it:

- Run this §4 script against **≥3 real trial users** (including at least one who
  bounced), transcripts in `docs/research/transcripts/`.
- A need that recurs across three of them with cited quotes becomes `confirmed`
  and graduates to [PROPOSALS.md](../PROPOSALS.md) as a ranked proposal with
  acceptance tests — it does **not** silently edit [DESIGN.md](../../DESIGN.md).
- The strongest lead to test first — because it is code-verified here, not just
  asserted — is the **onboarding cliff** (`wscore.go:87-106` dead-ends before
  `spawn`/`dashboard`): the cheapest fix with the largest first-hour effect.

_This artifact is versioned with the guide. When live adopter interviews land,
they replace these composite answers segment by segment — a claim here that a
real transcript refutes is retired, not defended._
