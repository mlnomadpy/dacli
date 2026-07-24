# Interview — Human operator

Status: **research transcript, segment = human operator.** This is one interview
answered firsthand, in the operator's own voice, against the script in
[`../INTERVIEW_GUIDE.md` § 3](../INTERVIEW_GUIDE.md). It is *evidence for one
source*, not a synthesis — a single transcript is `unverified` under § 9's trust
floor until a need recurs across ≥3 sources. Quotes are the unit of evidence; I
answer with the last time a thing actually happened, not with a wish (§ 0's
"don't lead" rule turned on myself).

**Who I am.** I run the swarm most days. I live in the CLI: `dacli spawn`,
`dacli agents --tail`, `dacli next`, `dacli integrate`. The dashboard is my
glanceable overview on the second monitor — I don't drive from it, I *watch*
from it. I think of dacli less as a task tracker and more as a loop I'm steering:
I set the goal, seed the backlog, and the governor keeps agents flowing until the
backlog is dry. My job is to keep that loop pointed at the right thing and to
catch the runs that go sideways before they burn a day. That framing matters for
everything below — I'm not managing tickets, I'm **driving a running system**,
and the questions I ask the dashboard are the questions you ask a dashboard in a
cockpit, not the ones you ask a spreadsheet.

---

## Warm-up / context

**1. Walk me through your last full day. When did you first open the dashboard, and what were you trying to learn?**

First thing, coffee in hand, before I spawn anything: I open the dashboard to
read the *shape* of yesterday. Not to do anything — to answer one question, "did
the loop make progress overnight or did it thrash?" I look at Burndown first —
done points per day — and at the Board counts: how many `done`, how many still
`blocked`. Yesterday the blocked count was stuck at 1 for the third day and that
told me something upstream was wedged, which is what I actually went and dug
into. The dashboard didn't tell me *what* was blocking what — I had to go read
`blocked/` and cross-reference by hand — but it told me *that* something was, and
that's what pulled me in. So the honest answer: I open it to get a delta, not a
detail. The details I still get from the CLI.

**2. How many agents do you usually have running at once? How do you keep track of them?**

Three to six. Past six I lose the thread. I keep `dacli agents --tail` running in
a dedicated pane — it's the single most-looked-at thing in my day, more than the
dashboard. I read it the way you read a heart monitor: I'm not reading the words,
I'm watching for the line that *stops changing*. A tail line that hasn't moved in
two minutes is my "something's wrong" trigger. The problem is `--tail` gives me
one line per agent — the last transcript line — and to actually know what an
agent is *doing* I have to leave the pane, find the run id, and open the
transcript file. So my "keeping track" is really two moves: a cheap glance that
tells me *which* agent to worry about, and an expensive dig that tells me *why*.
The gap between those two is where most of my day's friction lives.

---

## Observability (RQ1)

**3. Tell me about the last time you didn't know what an agent was doing. Walk me through the sequence.**

Two days ago. A `designer` agent on a docs task. `--tail` showed its last line
frozen on a `Read` of some architecture doc — no new line for maybe four
minutes. Sequence: (a) glanced at `--tail`, saw the frozen line; (b) ran `dacli
status` to check pending events and whether the run was even still alive — status
said the run was live, which was reassuring and useless, because "the process
exists" is not "the agent is making progress"; (c) opened the dashboard Swarm
panel hoping for more — it showed the same frozen last line, same pid, same
uptime ticking up, so no new information, just prettier; (d) gave up on the
tools and opened the raw transcript file, scrolled to the bottom, and saw it was
mid-`thinking`, actually fine, just chewing on a long doc. So it cost me four
tools and a context-switch to learn "it's thinking." The failure is that every
layer — `--tail`, `status`, Swarm — showed me the *same last line* re-skinned.
None of them distinguished **thinking** from **hung**. The raw transcript did,
because it had the `thinking` block; the projections threw that away.

**4. When you glance at the dashboard, what's the first thing your eye goes to? What do you trust most? Least?**

Eye goes to Burndown, the per-day bars — that's my "is the loop healthy" gauge.
Trust most: the Board `done` count, because it's downstream of `task done`, which
is downstream of an actual PR, so it's hard to fake. Trust least: the Swarm
panel's uptime and pid. Not because they're wrong — they're honestly probed — but
because they answer a question I don't have. I don't care that a pid exists; I
care whether the thing behind the pid is *advancing*. Uptime ticking up next to a
frozen last line is the dashboard looking busy while telling me nothing. So the
number I trust least is the one that's most alive-looking, which is its own kind
of trap.

**5. Last time the dashboard told you something was fine and it wasn't, or wrong and it was fine?**

The frozen-line case above is the "looked wrong, was fine" version — Swarm
implied stall, agent was thinking. The scarier direction, "looked fine, wasn't":
a run that had gone down a wrong *approach* — not stuck, not over budget,
happily committing — the dashboard showed it green and advancing, done points
even ticked up, and it was cheerfully building the wrong thing. Every gauge I
have measures *liveness and throughput*, none measures *correctness of
direction*. The dashboard can't show me "this agent is confidently wrong"
because nothing in the snapshot encodes intent-vs-approach. That's the blind spot
that actually costs me money — a hung agent wastes minutes, a confidently-wrong
one wastes a whole run and I only find it at PR review.

**6. What do you check that the dashboard doesn't show? What tab stays open next to it?**

Two tabs. One is the `--tail` pane (covered). The other is a terminal where I run
`dacli next` and, honestly, `git log --oneline` on the integration branch to see
what actually landed. The dashboard shows me counts; it doesn't show me the
*narrative* — "agent X shipped PR Y which closed task Z which unblocked W." I
reconstruct that story from git and `dacli integrate` output. The single thing I
check most that the dashboard never shows: **the dependency chain.** When
blocked count > 0, I want to see the graph. Today I read task files and build the
graph in my head. That's a workaround I do every single day.

---

## Steering (RQ2)

**7. Tell me about the last run you had to kill. How did you notice, how did you kill it, what did you wish had happened?** *(→ H2)*

Last week. Runaway budget — an implementer that got into a re-read loop,
re-discovering the same files a sibling had already mapped. I noticed it not from
the dashboard but from `--tail` scrolling *fast* — too many lines too quickly is
as bad a sign as no lines at all. To kill it: left the pane, ran `dacli agents`
to get the run ref, then `dacli kill <ref>`, waited for the SIGTERM→SIGKILL
grace, checked `killed.txt` landed. Maybe 40 seconds of me being a human router
between two terminals. What I wished: that the moment I saw it thrashing I could
have killed it *from where I was looking at it*. I was staring at the Swarm panel
watching it burn and the one thing I couldn't do was stop it from there. That's
the purest case for a write action — the observation and the intervention happen
in the same glance, and the tool forces me to teleport between them.

**8. Have you watched an agent go down the wrong path and wanted to say something — without killing it? What would you have said? What did you do instead?** *(→ H3 steer)*

Yes, and this is the one I have the most complicated feelings about. The wrong-
approach run from Q5 — I'd have loved to type one sentence: "you're solving the
generic case, the task only needs the claude-CLI path, narrow it." One sentence
would've saved the run. What I did instead: killed it, edited the task's
acceptance criteria to be sharper, re-spawned. That worked, but it cost a full
respawn and lost the context the agent had already built. So the *pain* is
absolutely real and recent. **But** — and this is me arguing against my own wish
— every time I've wanted to "chat" with an agent, the actual fix was that the
**brief was underspecified**. The steer I wanted to inject was information that
should have been in the acceptance criteria. If I make steering easy, I'll use it
to paper over bad briefs, and a swarm you have to babysit with live corrections
doesn't scale past the six agents I can watch. So: real pain, but I'm suspicious
that the *right* fix is a better brief and a `dacli ask`/escalation channel, not a
chat box. I'd rather the agent stop and ask me a sharp question than have me lean
over its shoulder narrating.

**9. When the backlog is in the wrong order, how do you fix priority today? How often?** *(→ H4)*

Rarely — maybe twice a week. I set `priority: must/nice` when I seed tasks and the
governor mostly does the right thing with `dacli next`. When I do reorder, I edit
the task's priority field or use the CLI. It's a chore but a small one, and it's
not *time-critical* — reprioritizing is a calm, deliberate act I do between
spawns, not a fire I'm fighting. That timing matters for the feature question:
drag-to-reorder would be *pleasant*, but I've never once been blocked by the lack
of it. It's the difference between friction I feel in the moment (killing,
steering) and friction I only notice when asked about it (reordering).

**10. Is there a moment you'd want to pause the whole thing — stop new spawns but not lose in-flight work? What triggers it?** *(→ H5)*

Yes, and more often than I'd expect. Trigger: I spot a systemic problem — a
vendor CLI flag drifted and three agents are all about to hit the same failure,
or I realize the goal I seeded was subtly wrong. I don't want to kill the four
healthy in-flight agents; I want to stop the governor from spawning *more* into a
known-bad situation while I fix the root cause, then resume. Today my only "pause"
is crude: I stop the loop process entirely, which also abandons the healthy runs,
or I let it keep spawning and clean up the mess after. A soft hold — "finish
what's running, spawn nothing new" — is something I've genuinely wanted in a
real, recent moment. It's the write action I'd most trust, because it's the least
invasive: it doesn't touch any agent's work, it just tells the *scheduler* to
wait.

**11. Walk me through the last PR you integrated. Where were you, what did you have open, would a dashboard gate have changed anything?** *(→ H6)*

Terminal. `dacli integrate` / `dacli merge --task NNN`, then GitHub for the final
look at the diff. The approval happens where the code is — in the diff view, in
git, with `verify` verdicts in front of me. A gate *button* in the dashboard
wouldn't have changed anything, because the dashboard doesn't show me the diff and
I would never approve a merge without reading the diff. If anything, a one-click
approve in a surface that *doesn't* show the code is a footgun — it invites
approving on vibes. Where the dashboard *could* help is showing me the **queue**:
which PRs have green `verify` verdicts and are waiting on me, so I know there's a
gate to go attend to. But the approval itself belongs where the evidence is, and
that's not the dashboard.

---

## Cost / trust (RQ3)

**12. If the dashboard could do things — kill, steer, reprioritize — would you trust it? What would make you not trust a button? Where do you want the friction of a terminal?**

I'd trust *destructive-but-reversible-in-spirit* actions there — pause, and even
kill, because kill is loud, audited (`killed.txt`), and its blast radius is one
run I was already watching. What I would *not* trust: any button that commits
something durable without showing me the evidence. Approve-a-PR without the diff.
"Steer" that silently rewrites what an agent is doing. The rule for me is: a
button is fine when the thing I'm looking at *is* the evidence for the decision
(the tail line thrashing → kill), and dangerous when the decision needs evidence
the surface isn't showing (the diff → approve). I *want* the terminal's friction
exactly where the action is irreversible and the dashboard can't show me enough
to be sure — merges, anything that edits history. The friction there isn't a bug,
it's the thing that makes me read before I act. So my verdict on the write
boundary isn't "never" — it's "write actions that operate on the *run* (kill,
pause) can live in the cockpit; write actions that operate on the *work* (merge,
edit content) must stay where the work is visible." That maps almost exactly onto
[DESIGN.md § 2](../../DESIGN.md)'s "runs agents, not work" line, which is why I
think it's a durable boundary and not just my preference.

---

## Spend & history (RQ1/RQ4)

**13. How do you know if a run is costing too much? When do you find out — during or after? What would early warning look like?** *(→ H7)*

Almost always *after*. I find out at calibration time, or when the governor's
window guard trips, or when I eyeball the run actuals post-hoc and wince. During
the run I have no burn *rate* — I have a cumulative number if I go look for it,
but not a "this run is spending 3× the calibrated band for its size" signal in
the moment. That's the gap. The runaway from Q7 — I caught it by the *proxy* of
tail scrolling fast, not by any actual spend signal, because there wasn't one in
front of me. Early warning I'd actually use: a burn-rate line per run against its
calibrated band, and — more than a chart — a **threshold that changes color when
a run crosses 1.5× its band.** I don't want to read a chart during a fire; I want
the chart to *yell*. The data's all there (governor windowSpent, run actuals,
the calibrated bands from the `role×model×runtime` work) — it's recorded and
never shown as rate-over-time. This is the read-only feature I'd get the most from
because overspend is my most expensive silent failure and it's the one the tool
already has every number to catch.

**14. After a run, how do you reconstruct what happened and when? Do you go back through events?** *(→ H8)*

Yes, and it's painful. The events are logged per day under `events/YYYY/MM/DD/`,
but reconstructing a *run's* story means correlating across that tree by hand —
which finding preceded which decision preceded which PR. I do it maybe once a
week, usually in a post-mortem when a run went wrong and I want to know the exact
turn it went wrong. A scrubbable timeline across a run would save me that
archaeology. It's not a moment-to-moment need — it's a retro need — so it's
lower-adrenaline than the burn warning, but when I need it I *really* need it and
the current answer is "grep the event log," which is a workaround, which per § 8
is the clearest map of an unmet need.

**15. Last time you wanted to see what blocks what. How did you figure out the dependency chain?** *(→ H1)*

Every day, per Q6. The critical path is *computed* — `internal/spm/criticalpath.go`
exists, the workspace knows the chain — and it is nowhere on the surface I look
at. When blocked count went to 1 and stuck, I opened task files, read their
`blocked-by` relationships, and drew the graph in my head to find the one task at
the root. That is the single most repeated manual reconstruction in my day. The
maddening part is it's a *read-only* feature — it doesn't cross any boundary, it
just draws a picture of data the workspace already computed. It's the cheapest
thing on the list to reconcile with the design and one of the highest-value to me,
which is a rare combination.

---

## Solution reaction — must / nice / no (RQ4, § 7)

Shown the mock-ups one at a time, after the problem talk. My reaction to each,
forced into must/nice/no, each tied to whether it maps to a *recent painful
moment* (strong) or just reads well (weak):

| # | Hypothesis | Verdict | Grounded in |
|---|---|---|---|
| **H1** | Task dependency / DAG view | **MUST** | Daily manual graph-in-my-head (Q6, Q15). Read-only, data already computed. Strongest evidence-to-cost ratio on the board. |
| **H7** | Budget / burn-rate charts (with a threshold that yells) | **MUST** | Overspend is my most expensive *silent* failure; today I catch it by proxy (Q13). Read-only, data already recorded. |
| **H5** | Pause / resume the loop (soft hold) | **MUST** | Recent, real: vendor-flag drift about to hit three agents (Q10). Least-invasive write action — touches the scheduler, not any agent's work. |
| **H2** | Cancel / kill from the UI | **NICE** | Real pain (Q7) but the CLI kill works and is only ~40s of friction. Worth it *only if* it lives right next to the tail/burn signal that triggers it — a kill button away from the evidence is pointless. |
| **H8** | Retro timeline | **NICE** | Real but low-frequency (Q14), a retro need not a live one. High value the ~once/week I need it; not a daily driver. |
| **H3-view** | Richer live transcript *view* (see the `thinking`, not just the last line) | **NICE→MUST-adjacent** | Directly fixes the thinking-vs-hung blind spot (Q3, Q4). This is the read-only half of H3 and I'd take it fast. |
| **H3-steer** | Inject guidance / chat mid-run | **NO (as a chat box)** | Pain is real (Q8) but the honest fix is a better brief + a `dacli ask` escalation channel. A chat box scales to the six agents I can babysit and no further; it rewards bad briefs. Refute the *feature*, keep the *need*. |
| **H4** | Drag to re-prioritize | **NO** | Never once blocked me (Q9). Pleasant, not needed. Pure mock-up appeal, weakest evidence class. |
| **H6** | Approve-gate a PR from the UI | **NO (approval); NICE (queue)** | Approval belongs where the diff is (Q11) — a one-click approve without the code is a footgun. But *showing the queue* of verdict-green PRs waiting on me is a fine read-only add. |

**Forced trade-off — if you could ship only two next quarter, which two, what do you give up?**
**H1 (DAG view) and H7 (burn-rate with a threshold that yells).** I give up
everything write-side, including H2 and H5, without much pain — because both of my
picks are read-only, both attack my two most-repeated daily costs (blind
dependency reconstruction, silent overspend), and both are cheap precisely
*because* they don't touch the write boundary. If I got a third for free it'd be
**H5 (pause)**, the one write action I fully trust.

---

## The top 3 I would pay for

Ranked by recent, painful, past-behavior evidence — not by demo appeal:

1. **H7 — Burn-rate visibility with a threshold that yells.** My most expensive
   *silent* failure is a runaway I catch late or by proxy. Every number needed is
   already recorded (governor `windowSpent`, run actuals, calibrated
   `role×model×runtime` bands); it just isn't shown as rate-over-time with an
   alert. This turns a full wasted run into a 30-second catch. Highest dollar
   impact, read-only, no boundary cost.
2. **H1 — Dependency / DAG view.** The single most repeated manual
   reconstruction in my day (I build the blocked-graph in my head every time
   `blocked > 0`). The critical path is *already computed* in
   `internal/spm/criticalpath.go` and shown nowhere. Cheapest to reconcile with
   the read-only design, near-daily payoff.
3. **H5 — Pause / resume the loop.** The write action I'd trust most and have
   wanted in a real, recent moment (systemic problem about to hit multiple
   in-flight agents). It operates on the *scheduler*, not on any agent's work, so
   it respects the "runs agents, not work" boundary — the safest hole to punch in
   the read-only projection.

Everything below these is either genuinely nice (H2, H8, H3-view) or something I'd
argue you *shouldn't* build as drawn (H3-steer as chat, H4, H6-approval).

---

## Open floor — un-hypothesized needs (RQ5, § 8)

*The most important section, per § 8. A need that recurs across three interviews
here outranks any hypothesis above.*

- **"Thinking vs. hung" is a first-class signal you don't expose.** Every
  observability tool I have (`--tail`, `status`, Swarm) collapses to the same
  *last line*, and none distinguishes an agent that's mid-`thinking` from one
  that's wedged. I burn context-switches resolving that ambiguity several times a
  day (Q3). The transcript knows — it has the `thinking` block — and the
  projections throw it away. If you gave me *one* new field it'd be a per-agent
  state: `thinking | acting | waiting | stalled`, honestly derived. That's worth
  more to me than most of the eight hypotheses and it isn't on the list.

- **"Confidently wrong" has no gauge.** My gauges all measure liveness and
  throughput; none measures whether an advancing agent is advancing in the
  *right direction* (Q5). A confidently-wrong run is my costliest failure and I
  only catch it at PR review. I don't know the shape of the fix — maybe it's
  surfacing the agent's stated approach early enough that I can sanity-check it
  against the acceptance criteria before it's spent a run. But the *need* — an
  early "is this even the right approach" checkpoint — is real and unhypothesized.

- **Escalation, not chat.** The steering pain (Q8) keeps resolving to the same
  root: I want the agent to **stop and ask me a sharp question** when it hits a
  fork, rather than me leaning in to narrate. `dacli ask` exists as a channel;
  what's missing is it surfacing to *me, live, in the cockpit* — "agent X is
  blocked on a question, here it is." That's the steering primitive I'd actually
  build instead of H3-chat: pull, not push. It scales; a chat box doesn't.

- **The workaround that maps the biggest gap:** I keep a terminal open running
  `git log` on the integration branch to reconstruct the *narrative* of what
  landed and what it unblocked, because the dashboard shows counts, not story
  (Q6, Q14). The habit of dropping to git for the story is the clearest evidence
  that the surface is missing a causal, "X shipped → closed Y → unblocked Z" view
  — which is really H1 (DAG) and H8 (timeline) being the *same* underlying need
  seen from two angles: the swarm's dependency structure over *space* and over
  *time*.

- **Who else to talk to:** the operator running dacli *headless on a schedule*,
  who isn't watching `--tail` at all. Everything I've said is framed by "I'm
  watching six agents I can see." The person who lets it run overnight has the
  opposite problem — they need the *asynchronous* version of every signal here
  (a digest, an alert that reaches them off-console), and they'd rank H7's
  yelling-threshold and the escalation channel even higher than I do, because for
  them there's no `--tail` pane to catch anything by proxy.

---

_One transcript, one segment, `unverified` until it recurs. The three findings I'd
carry into synthesis with the most confidence — because they're grounded in
daily, past-behavior stories, not mock-up reactions — are: **burn-rate needs to
yell (H7), the dependency graph is a daily manual reconstruction (H1), and
"thinking vs. hung" is an unmet first-class observability need.** Feed confirmed
needs to [PROPOSALS.md](../PROPOSALS.md); the write-boundary verdict I'd defend is
"operate-on-the-run actions (pause, kill) can enter the cockpit; operate-on-the-
work actions (merge, edit) stay where the work is visible."_
