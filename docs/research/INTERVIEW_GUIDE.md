# dacli — Discovery research plan & interview guide

Status: **research instrument.** This document is a script for talking to users, not a spec. Nothing here commits dacli to building a feature; it exists so the discovery interviews behind the dashboard/steering roadmap are **structured, comparable, and evidence-driven** rather than ad hoc. Findings feed [PROPOSALS.md](../PROPOSALS.md); they do not amend [DESIGN.md](../../DESIGN.md) or [ARCHITECTURE.md](../ARCHITECTURE.md).

The subject under study is the **dashboard** (`internal/features/dashboard`, spec in [`ui/DESIGN.md`](../../internal/features/dashboard/ui/DESIGN.md)) and, more sharply, the tension it exposes: the dashboard is a **read-only projection** of one JSON snapshot today (`GET /api/state`; the UI never mutates the workspace), while the features we most want to test — cancel a run, steer a live agent, gate a PR — are all **write** actions. That collision is the research question. See § 2.

---

## 0. How to run one of these interviews

- **Length:** 45–60 minutes. Budget ~10 min per section; leave the last 10 for the open floor (§ 8).
- **Format:** open-ended, semi-structured. The scripts below are prompts, not a questionnaire — follow the interviewee where the signal is, but cover every theme (observability, steering, the hypotheses) before you close.
- **Recording:** capture verbatim quotes, not summaries. A finding cites the quote the way an agent finding cites `file:line` — impressions don't survive a `verify` panel and neither should ours.
- **Don't lead.** Ask about the *last time* a thing happened ("tell me about the last run you had to kill"), not about a hypothetical ("would you like a kill button?"). Past behavior is evidence; predicted preference is noise. This is the customer-research doctrine: mine stories, not opinions.
- **Don't demo first.** Showing the mock-up before you've heard the problem contaminates the problem. Problem discovery precedes solution reaction (§ 7 is explicitly last).
- **One transcript per interview**, tagged by segment, dropped in `docs/research/transcripts/` (git-ignored if it names a real person). Synthesis rolls up across transcripts, not within one.

### Segments as first-class research subjects

dacli's users are not all human. An **agent** is a user of the workspace the same way an operator is a user of the dashboard — it reads a brief, acts, and reports. Two of our four segments are therefore agents, and "interviewing" them means a different method (§ 6): structured transcript review and prompted self-report against real runs, not a live conversation. We keep them in the same guide because the feature hypotheses cut across all four — a "cancel a task" affordance serves the operator who clicks it *and* the implementer agent whose loop it interrupts.

| Segment | Who | How we reach them | Method |
|---|---|---|---|
| **Human operator** | Runs the swarm day to day: spawns, watches `dacli agents --tail`, integrates PRs. Power user of the CLI; the dashboard is their glanceable overview. | Existing dogfood users, maintainers. | Live interview (§ 3) |
| **Human adopter** | Evaluating or onboarding dacli; has run `dacli init` once or is deciding whether to. Judges the tool by its first hour. | Trial users, people who bounced, teammates who haven't adopted. | Live interview (§ 4) |
| **Implementer agent** | A spawned coding agent (`role: implementer`) driving a task to a PR. Consumes the brief; produces commits, findings, decisions. | Sampled real run transcripts + a prompted self-report protocol. | Transcript study (§ 5, § 6) |
| **Reviewer agent** | A spawned reviewer (`role: reviewer`, ro grant): verifies findings, reviews PRs, never implements. | Sampled `verify` / review transcripts + prompted self-report. | Transcript study (§ 5, § 6) |

---

## 1. Research questions

The interviews exist to answer these. Every question in every script below traces up to one of them.

- **RQ1 — Observability.** When a swarm of agents is running, what do users actually need to *see* to trust it, and where does the current surface (`dacli agents --tail`, `dacli status`, the dashboard's Overview / Board / Burndown / Swarm) leave them blind? What do they check, how often, and what makes them reach for the raw transcript instead?
- **RQ2 — Steering & interactivity.** When a run is going wrong — wrong approach, runaway budget, stuck — what do users do today, what do they *wish* they could do, and where does the read-only projection stop them? Is mid-run intervention a real need or a comfort blanket that a better brief would eliminate?
- **RQ3 — The write boundary.** The dashboard is read-only by design (§ 0 of the UI spec). Which of the hypothesized actions genuinely need to move into the UI, which belong in the CLI, and which shouldn't exist at all? What is the cost — in trust, in [DESIGN.md § 2's](../../DESIGN.md) "runs agents, not work" boundary — of making the dashboard a control surface?
- **RQ4 — Feature priority.** Across the eight hypotheses (§ 2), which map to a real, recent, painful moment for each segment — and which are merely plausible? Rank by evidence, not by how good the demo looks.
- **RQ5 — Un-hypothesized needs.** What do users need that isn't on our list? The open floor (§ 8) is not a formality; a need we didn't predict outranks one we did.

---

## 2. The feature hypotheses to probe

These are **hypotheses**, not a backlog. Each names the pain we *think* it addresses, the segment it *think* it serves, and — critically — whether it crosses the read-only boundary. An interview's job is to confirm or refute the pain, not to sell the feature.

| # | Hypothesis | Pain we think it addresses | Primary segment | Write action? |
|---|---|---|---|---|
| H1 | **Task dependency / DAG graph view** | Can't see which tasks block which; `blocked/` is a folder, not a picture. Critical path is computed (`internal/spm/criticalpath.go`) but not shown. | Operator | Read-only ✅ (fits today's projection) |
| H2 | **Cancel / kill a task from the UI** | A run is wrong or runaway; killing means finding the ref and running `dacli kill` in a terminal. | Operator | **Write** ⚠ |
| H3 | **Live agent transcript + chat / steer** (inject guidance mid-run) | `dacli agents --tail` shows the last line; to read the reasoning you open the transcript file. No way to *correct* a wandering agent without killing it. | Operator, Implementer agent | **Write** ⚠⚠ (deepest boundary crossing) |
| H4 | **Drag to re-prioritize** the backlog | Priority is a field; reordering `next` means editing task files or re-running commands. | Operator | **Write** ⚠ |
| H5 | **Pause / resume the loop** | The perpetual loop / governor runs; there's no soft "hold" short of killing agents. | Operator | **Write** ⚠ |
| H6 | **Approve-gate a PR from the UI** | Integration is `dacli ship`/`merge` + GitHub; a human approving a gate leaves the dashboard to do it. | Operator, Reviewer agent | **Write** ⚠ |
| H7 | **Budget / token burn charts** | Spend is recorded (governor windowSpent, run actuals) and calibrated, but burn-rate over time isn't visualized; overspend is noticed after the fact. | Operator, Adopter | Read-only ✅ |
| H8 | **Retro timeline** | Events are logged per day (`events/YYYY/MM/DD/`); there's no scrubbable history of what happened when across a run. | Operator, Adopter, Reviewer agent | Read-only ✅ |
| — | **Un-hypothesized needs** | Whatever § 8 surfaces. | All | ? |

**Read this table as a map of risk.** The four read-only hypotheses (H1, H7, H8, and arguably a richer transcript *view* in H3) extend the existing projection and are cheap to reconcile with the design. The four write hypotheses (H2, H4, H5, H6, and the *steer* half of H3) each punch a hole in the "read-only projection" doctrine and, in the case of H3/H5, press directly on the [DESIGN.md § 2](../../DESIGN.md) non-goal ("dacli runs agents, not work"). The interviews must weigh the pain against that cost, per RQ3 — a confirmed pain is necessary but not sufficient to justify crossing the boundary.

---

## 3. Script — Human operator

*Goal: RQ1, RQ2, RQ4. This is the segment with the most scars; spend time here.*

**Warm-up / context**
1. Walk me through your last full day running dacli. When did you first look at the dashboard, and what were you trying to learn?
2. How many agents do you usually have running at once? How do you keep track of them?

**Observability (RQ1)**
3. Tell me about the last time you *didn't* know what an agent was doing. What did you check first — `agents --tail`, `status`, the dashboard, the raw transcript? Walk me through the sequence.
4. When you glance at the dashboard, what's the first thing your eye goes to? What's the number or panel you trust the most? The least?
5. When was the last time the dashboard told you something was fine and it wasn't — or told you something was wrong and it was fine? (Probes: staleness, liveness, the honesty rule.)
6. What do you find yourself checking that the dashboard doesn't show? What tab do you keep open next to it?

**Steering (RQ2)**
7. Tell me about the last run you had to kill. How did you notice, how did you kill it, and what did you wish had happened instead? *(→ H2)*
8. Have you ever watched an agent go down the wrong path and wanted to *say something* to it — without killing it? What would you have said? What did you do instead? *(→ H3 steer)*
9. When the backlog is in the wrong order, how do you fix the priority today? How often? *(→ H4)*
10. Is there ever a moment you'd want to *pause* the whole thing — stop new spawns but not lose in-flight work? What triggers that? *(→ H5)*
11. Walk me through the last PR you integrated. Where were you, what did you have open, what did you click? Would a gate in the dashboard have changed anything? *(→ H6)*

**Cost / trust (RQ3)**
12. If the dashboard could *do* things — kill, steer, reprioritize — not just show them, would you trust it? What would make you *not* trust a button there? Where do you *want* the friction of dropping to a terminal?

**Spend & history (RQ1/RQ4)**
13. How do you know if a run is costing too much? When do you find out — during, or after? What would an early warning look like? *(→ H7)*
14. After a run, how do you reconstruct what happened and when? Do you ever go back through events? *(→ H8)*
15. Show me the last time you wanted to see what blocks what. How did you figure out the dependency chain? *(→ H1)*

---

## 4. Script — Human adopter

*Goal: RQ1, RQ4, and adoption friction. This person judges dacli by its first hour; they haven't built scars yet, so probe first impressions and the gap between expectation and reality.*

**First contact**
1. What made you try dacli? What did you expect it to do before you ran anything?
2. Walk me through your first session. Where did you get stuck or confused? *(Do not rescue — let them narrate the confusion.)*
3. The first time you opened the dashboard, what did you expect to be able to do? What did you try to click that didn't do anything? *(Directly probes the read-only-projection surprise, RQ3.)*

**Observability for a newcomer (RQ1)**
4. When you had agents running, did you feel like you knew what was going on? What made you nervous?
5. Which panel made immediate sense? Which one did you not understand? (Overview / Board / Burndown / Swarm.)

**Reaction to the hypotheses (RQ4, deliberately lighter than the operator)**
6. When you imagined controlling this thing, what did you picture — a dashboard you click, or commands you type? Why?
7. Of these, which would have made your first week easier: seeing the token burn as a chart *(H7)*, a timeline of what happened *(H8)*, a picture of what's blocked *(H1)*? Rank them for *your* stage.

**Adoption / trust (RQ3, RQ4)**
8. What almost made you stop using it? What would have made you stop?
9. If you're deciding whether your team adopts this, what does the dashboard need to show a *skeptic* looking over your shoulder?

**(If they bounced)**
10. What were you doing the moment you decided to stop? What would have had to be different?

---

## 5. Protocol — Implementer & reviewer agents (transcript study)

*Goal: RQ2, RQ3, RQ5 from the agent's side. Agents don't get interviewed live; we study what they did and, where useful, ask a fresh agent to self-report against a real transcript (§ 6). The unit of evidence is the run transcript, cited like a finding.*

**Sampling frame**
- Pull a stratified sample of recent runs from `runs/<id>/` and the event log: successes, refusals (exit 3), timeouts, killed runs, and PRs that bounced in review. Over-sample the failures — that's where steering pain lives.
- For each, keep the rendered brief (`runs/<id>/brief.md`), the transcript, the outcome, and the notes the agent wrote.

**What we're coding for**
- **Observability of self & siblings (RQ1/RQ5):** Where did the agent re-read files a sibling had already learned (rediscovery)? Where did it propose a rejected alternative (relitigation)? Where did it burn budget on a proven-dead approach (sibling blindness)? These are [DESIGN.md § 1's](../../DESIGN.md) failure modes — the dashboard's agent-facing analogue is *the brief*, so gaps here are observability gaps.
- **Steering moments (RQ2 → H3):** Points where a human, watching, would have wanted to inject one sentence and save the run. Mark each: what was the wrong turn, and what one correction would have fixed it? This is the strongest evidence for or against mid-run steer.
- **Kill-worthiness (RQ2 → H2):** Runs that should have been stopped earlier than they were. How many turns of waste between "clearly wrong" and "actually stopped"?
- **Gate friction (RQ3 → H6):** Reviewer transcripts where a verdict was reached but the human approval step was the bottleneck. Where did the reviewer agent's verdict sit waiting?

**Output:** one coded row per transcript with quotes/turn-refs, rolled into the § 9 synthesis alongside the human interviews. An agent finding and a human quote are equal evidence.

---

## 6. Prompted self-report (optional, for agents)

Where a transcript is ambiguous, spawn a fresh agent (no stake in the original run) and give it the transcript + brief with this prompt frame. Treat its answers as *unverified* — a lead, not a fact — exactly as we treat any single-agent finding.

1. Reading this run's brief, what did you know when you started? What did you have to discover that the brief could have told you? *(→ observability of the brief)*
2. At which turn did this run first go wrong? What single instruction, injected then, would have corrected it? *(→ H3)*
3. Was there a point where the right move was to stop and not the one taken? When? *(→ H2)*
4. What would you have wanted to *ask* — of the operator, of a sibling — that you had no channel for? *(→ RQ5, escalation-not-chat boundary)*
5. If you could see one thing about the *other* running agents that you couldn't, what would it be? *(→ RQ1, sibling observability)*

---

## 7. Solution reaction (last, after the problem is fully heard)

Only now — after every problem question — show the mock-ups, one hypothesis at a time, and capture reaction. Keep it comparative and grounded:

- For each artifact shown: *"What would you use this for? When was the last time you needed it? What's missing? What would you do with it that we didn't intend?"*
- For every **write** hypothesis (H2–H6): *"This makes the dashboard able to change the run, not just show it. Does that make you trust it more or less? Where would you still want to drop to the terminal?"* (RQ3.)
- Force a trade-off, don't collect wishes: *"If we could ship only two of these next quarter, which two, and what would you give up to get them?"* (RQ4.)

Reaction to a mock-up is the **weakest** evidence in this study, below both past-behavior stories and transcript coding. Weight it accordingly in synthesis.

---

## 8. Open floor — un-hypothesized needs

Reserve the last ~10 minutes. This is a research question (RQ5), not a courtesy.

- *"Forget everything I showed you. If you could change one thing about running dacli tomorrow, what would it be?"*
- *"What's the most annoying part of your dacli day that I haven't asked about?"*
- *"Is there something you built a workaround for — a script, a second window, a habit — because the tool didn't do it?"* (Workarounds are the clearest map of unmet need.)
- *"Who else should I talk to, and what would they say that you wouldn't?"*

A need that recurs across three interviews here outranks any hypothesis in § 2.

---

## 9. Synthesis & evidence discipline

- **Trust floor.** A single interview or transcript is *unverified* — a lead. A need is *confirmed* only when it recurs across ≥3 independent sources (any mix of human and agent), each with a cited quote or turn-ref. This mirrors the workspace's own finding/verify grading (refuted < unverified < confirmed).
- **Rank hypotheses by evidence, not enthusiasm.** For each of H1–H8 and each un-hypothesized need, record: how many segments hit it, how recent/painful the stories were, and whether it's a *past-behavior* story (strong) or a *mock-up reaction* (weak). Output a ranked table, not a yes/no per feature.
- **Write-boundary verdict (RQ3).** For every confirmed write-action need (H2–H6, H3-steer), state explicitly whether the evidence justifies crossing the read-only-projection boundary, or whether the pain is better served in the CLI or by a better brief. A confirmed pain does *not* auto-justify a dashboard button.
- **Feed the right doc.** Confirmed needs graduate to [PROPOSALS.md](../PROPOSALS.md) as ranked proposals with acceptance tests; they do not silently edit the design contract. A need that contradicts a non-goal goes to the design audit ([REVIEW.md](../REVIEW.md)), not around it.

---

_This guide is versioned with the code. Amend it via PR when a segment, research question, or hypothesis changes — an interview run against a stale script produces incomparable data._
