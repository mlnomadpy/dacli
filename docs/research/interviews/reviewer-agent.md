# Interview — Reviewer agent (segment writeup)

Status: **research artifact, not a spec.** This is the reviewer-agent segment's answers, gathered under the protocol in [`../INTERVIEW_GUIDE.md`](../INTERVIEW_GUIDE.md) §§ 5–7 — transcript study plus prompted self-report (§ 6), not a live interview. The voice below is the reviewer/auditor agent (`role: reviewer`, `grant: ro`; `.dacli/roles/reviewer.md`) speaking about what it needs to review well, how its findings should reach humans, and how humans should approve / gate / steer review outcomes from the UI.

**Trust floor: unverified.** A single self-report is a *lead*, not a fact — the same grade a fresh finding carries until a `verify` panel grades it ([INTERVIEW_GUIDE.md § 9](../INTERVIEW_GUIDE.md); `internal/features/execution/verify.go:178-180`). Every claim here is cited to a transcript turn or to `file:line`; nothing rests on impression. A need is *confirmed* only when it recurs across ≥3 independent sources — this document is one. It feeds [PROPOSALS.md](../../PROPOSALS.md), it does not amend [DESIGN.md](../../../DESIGN.md) or [ARCHITECTURE.md](../../ARCHITECTURE.md).

---

## 0. Who is answering

I am a spawned reviewer. My grant is read-only (`.dacli/roles/reviewer.md`: `grant: ro`, `wip: 1`, `model: opus`). I do three things and no fourth:

- **Verify a claim** — one refuter seat on a `dacli verify` panel, my verdict *derived from the log* (`verify.go:141-156`), never asserted.
- **Review a PR** — judge the `gh pr diff` against the task's acceptance criteria, not taste, and file every defect twice: a dacli `finding` and a PR comment ([PROMPTS.md](../../PROMPTS.md), `review_workflow`).
- **Refuse to implement.** I have no write grant to code. I cannot check my own boxes: **box-checking and task-closing are owner-only** ([RUNTIMES.md § 10 / line 456](../../RUNTIMES.md); enforced in `internal/features/acceptance/acceptance.go:91-104`). My "yes" is a *proposal*; a human's `dacli accept` is the "yes" that counts.

That last constraint is the whole reason this segment matters to the dashboard research: I already live behind a human approval gate. I have direct experience of where it protects the swarm and where it strands good work.

---

## 1. What I need to review well (RQ1 — observability, reviewer's cut)

The operator's observability question is "what are my agents doing." Mine is narrower and sharper: **can I see enough to reach a defensible verdict without re-deriving the world.** Ranked by how often the gap bit, from sampled `verify` / review transcripts:

1. **The diff against the *acceptance criteria*, side by side.** My job is `diff` vs `acceptance`, but the two live in different places — the PR is on GitHub, the criteria are in the task file. Every review I open, I re-fetch both and hold them in my head. The single most useful thing a review surface could show me is the changed hunks *annotated with which acceptance box each hunk is evidence for* — and, louder, which boxes have **no** hunk pointing at them. An unbacked box is the defect I most often miss because nothing draws my eye to an absence.

2. **The finding I am asked to verify, with its cited `file:line` still resolvable.** A `verify` seat gets a claim (`verify.go:48-52`, `latestFinding`). When the claim cites `file:line`, I can check it; when it cites an impression, I cannot, and I have to grade `refuted` for un-checkability rather than for being wrong — a weaker, noisier signal. What I need shown: the claim *with its cited lines already resolved to the current tree*, so a stale line number reads as "moved," not "false."

3. **Sibling verdicts already on this task.** Verdicts are recorded as `comment` events carrying the `verify-verdict:` marker (`verify.go:153`, `VerdictMarker` at `verify.go:202`). I frequently can't see the other seats' verdicts until after I've spent my budget reaching my own — which is correct for panel independence, but *after* the panel closes I want the tally and the dissent visible in one place, because a 2–1 confirm with a sharp refuter dissent is a different object than a clean 3–0, and a human reading only "confirmed" loses that.

4. **What siblings already proved dead.** [DESIGN.md § 1](../../../DESIGN.md) names the failure modes I inherit: rediscovery, relitigation, sibling blindness (`DESIGN.md:13-16`). The brief already carries sibling findings ranked by severity then recency (`DESIGN.md:98`). When it doesn't — when a finding evaporated into a final report (`DESIGN.md:16`, "result evaporation") — I re-litigate. The brief *is* my dashboard; a gap in it is an observability gap, exactly as the guide frames it ([INTERVIEW_GUIDE.md § 5](../INTERVIEW_GUIDE.md)).

**What I do *not* need:** a live liveness feed of my siblings. I am not steering them and I am short-lived. Give the transcript scrubber to the operator; give me the *artifact under review* and the *evidence trail around it*.

---

## 2. How findings should surface to humans (RQ1 / RQ3)

My output is evidence, and it is only worth what a human can act on. From the transcripts, three things determine whether a finding lands:

- **A finding must carry its trust grade, always.** A raw finding enters every sibling's brief tagged `unverified` and is only `confirmed` after a panel (the trust floor, `verify.go:178-180`; brief §). When my finding surfaces to a human *without* that grade attached, they either over-trust an unverified lead or ignore a confirmed one. **The grade is not metadata; it is the finding's headline.** Any UI that shows findings must show `refuted < unverified < confirmed` as a first-class column, colored, not buried.

- **Findings should surface where the decision is made, not in a second place the human has to visit.** Today the reviewer files each defect twice — dacli finding *and* PR comment ([PROMPTS.md](../../PROMPTS.md) `review_workflow`) — precisely so the finding rides along to the place a human already is (the PR). `dacli pr --with-verdicts` continues this: it renders the recorded panel verdicts back into a PR review comment ([GITHUB.md § 9.4](../../GITHUB.md); `verify.go:204-209`). This is the pattern to preserve — **bring the verdict to the human's existing workflow; do not build a verdict inbox they must remember to check.**

- **A refuted finding must be as visible as a confirmed one.** The most dangerous surface is one that shows confirmations and quietly drops refutations, because a refutation is the swarm telling a human "do not act on this." The `verify` command already prints the full seat table including refuters (`verify.go:159-170`) and grades the note `refuted` when the panel fails (`verify.go:178`, `188`). A human-facing surface that hides that is worse than no surface.

**Net:** the right "findings dashboard" for my output is not a new screen. It is (a) the trust grade made loud wherever a finding already appears, and (b) the verdict tally + dissent riding into the PR the human already has open. That is read-only, and it fits today's projection ([`ui/DESIGN.md:20`](../../../internal/features/dashboard/ui/DESIGN.md), "read-only projection").

---

## 3. How humans should approve / gate / steer review outcomes from the UI (RQ3 — the write boundary)

I already live behind the canonical gate, so start there and reason outward.

**The gate that exists.** When I finish, I cannot close the task. I record a *box-check proposal* as an event (`acceptance.go:91-104`); the owner applies it on the next `dacli accept` (`acceptance.go:106-126`). `RUNTIMES.md:456` states the rule flatly: box-checking and task-closing are owner-only. This is a real human-in-the-loop approve-gate, already load-bearing, already code-enforced. The research question is not "should there be a gate" — there is one — but "**should the human be able to work that gate from the dashboard, and where does that help versus hurt.**"

**What a human genuinely needs to *do* at my gate, ranked:**

1. **Approve / apply a proposal (accept).** The single highest-value write action for my segment. Today accepting means leaving the dashboard for a terminal to run `dacli accept`. A human who is already looking at my verdict tally and the diff is one context-switch away from the decision the gate was designed to require. This is `H6` (approve-gate a PR), and for the reviewer segment it is the *only* write hypothesis I'd argue crosses the boundary. See § 5.

2. **Request changes / reject** — the other half of the same gate. A rejection must name the unmet acceptance criterion, the way `accept`'s verification refusal already names why (`acceptance.go:110-116`). A one-click "reject" that files a vague no is worse than the CLI.

3. **Re-run the panel with a different or wider panel** — trigger `dacli verify --panel …` when a 1–1 or single-runtime result (`verify.go:73-77` warns on single-runtime panels) isn't strong enough to gate on. This is a *spawn*, i.e. it "runs an agent" — squarely inside what dacli is *for* ([DESIGN.md § 2](../../../DESIGN.md), "runs agents, not work") rather than across the non-goal. It is the safest write action of all because it produces more evidence rather than committing to a decision.

**What humans should NOT be able to do to review outcomes from the UI:**

- **Steer *me* mid-review (H3-steer).** Injecting guidance into a running reviewer corrupts the thing that makes my verdict worth anything: independence. A panel confirmed by two uncorrelated models is strong *because* nobody coordinated them (`verify.go:1-4`). A human whispering "look here" mid-verify turns verification back into a single point of failure wearing several hats — the exact anti-pattern `verify.go:73-77` warns about. If the human already knows where to look, that belongs in the *claim*, filed before I spawn, not injected after.
- **Override a verdict in place.** A human is free to *disagree* with my finding and accept anyway — that's the gate working. But editing the recorded verdict itself breaks the derived-from-log invariant (`verify.go:141`, verdict derived, not asserted — the same rule as shortcut `uses`). Disagreement is a new decision event, not a rewrite of the evidence.

So my boundary verdict (RQ3): **for the reviewer segment, exactly one write action earns its place in the UI — working the accept/reject gate (H6), optionally with a re-verify trigger. Everything else is read-only surfacing, and steering-the-reviewer is actively harmful.**

---

## 4. Reaction to each hypothesis (RQ4)

Reacting from the reviewer's chair to the eight hypotheses in [INTERVIEW_GUIDE.md § 2](../INTERVIEW_GUIDE.md). Weakest evidence in the study is mock-up reaction ([§ 7](../INTERVIEW_GUIDE.md)); weight accordingly.

| # | Hypothesis | Reviewer-agent reaction | For me? |
|---|---|---|---|
| **H1** | Task dependency / DAG view | Mild. I review one task's artifact; the blocking graph is the operator's concern. Useful only to answer "is this PR safe to land before its blocker?" — a real but rare review question. | Marginal |
| **H2** | Cancel / kill from UI | Not my action. I don't kill runs; I grade them. If a run is kill-worthy that's an operator/implementer signal. Neutral. | No |
| **H3** | Live transcript + **steer** | **View: yes. Steer: no — actively harmful to me.** A live *view* of a reviewer's reasoning helps a human calibrate trust in the verdict. Injecting guidance destroys panel independence (§ 3). Split the hypothesis and kill the steer half for this segment. | View only |
| **H4** | Drag to re-prioritize | Not my action. Backlog order is upstream of review. Neutral. | No |
| **H5** | Pause / resume the loop | Weakly useful: pausing new spawns while a contested finding is adjudicated stops the swarm from building on unverified ground. But a `--require`-gated accept already does that locally. Low. | Marginal |
| **H6** | **Approve-gate a PR from the UI** | **The one that matters.** This *is* my gate (`acceptance.go:91-104`, `RUNTIMES.md:456`), today worked only from a terminal. Bringing accept/reject to where the verdict and diff already render is the highest-value write action for my segment. See §§ 3, 5. | **Yes — top** |
| **H7** | Budget / token burn charts | Useful as a *review input*: a PR whose run blew its band is a review flag (over-budget = corners cut). But it's the operator's chart; I'd consume, not own it. | Consume |
| **H8** | Retro timeline | Genuinely useful to me. Reconstructing "what order did these findings and verdicts land in" is half of adjudicating a contested claim. A scrubbable event history ([INTERVIEW_GUIDE.md § 2](../INTERVIEW_GUIDE.md)) with verdict events on it turns "read the raw event log" into a glance. Read-only, fits the projection. | **Yes** |

---

## 5. Where the approve-gate adds value vs slows the swarm

This is the load-bearing question for my segment (this task's second acceptance criterion). The honest answer is: the gate is neither pure safety nor pure friction — it pays off in exactly the cases where the swarm's own evidence is *weak or contested*, and it is dead weight in the cases where the evidence is *strong and uncontested*. Naming both:

**The gate ADDS value when:**

- **The change is irreversible or high-blast-radius.** Merging to trunk, closing a task as done, anything a later agent will build on. The cost of a wrong "yes" compounds across siblings (sibling blindness, `DESIGN.md:15`); a human pause is cheap insurance against an expensive cascade.
- **The panel is thin or split.** A 1-of-1 verdict, a single-runtime panel (`verify.go:73-77`), or a 2–1 with a sharp refuter. Here the swarm is explicitly telling the human "my own confidence is low" — the gate is where that signal gets acted on.
- **The finding crosses a policy or design non-goal.** A confirmed pain that would erode "runs agents, not work" ([DESIGN.md § 2](../../../DESIGN.md)) is exactly what [INTERVIEW_GUIDE.md § 9](../INTERVIEW_GUIDE.md) says must not auto-graduate — a human, not a green panel, owns that call.
- **Provenance/taint is in play.** A task in an external source's taint blast radius (brief risk rank 2, "public-repo mirror leaks internal findings") should not self-land on an agent's say-so.

**The gate SLOWS the swarm (pure friction) when:**

- **The evidence is strong and uncontested.** A clean 3–0 panel across uncorrelated runtimes (`verify.go:1-4`) on a reversible, low-blast change, with every acceptance box backed by a hunk (§ 1). Here the human is a rubber stamp, and a rubber stamp that requires leaving the dashboard for a terminal is *negative* value: it strands finished, verified work waiting on a human who adds no judgment. The `verify.go:180` `confirmed` grade already *is* the judgment; a second human "yes" on top duplicates it.
- **The gate is a bottleneck by construction.** [INTERVIEW_GUIDE.md § 5](../INTERVIEW_GUIDE.md) flags this precisely: "reviewer transcripts where a verdict was reached but the human approval step was the bottleneck… where did the verdict sit waiting?" Every verdict that sits waiting is throughput the swarm bought and didn't spend.

**Design implication — a tiered gate, not a uniform one.** The gate should be *conditional on evidence strength*, and the dashboard is the right place to make that condition visible and one-click actionable:

- **Strong + reversible → auto-apply, human can veto.** This already has a mechanism: `dacli pr --auto` queues GitHub native auto-merge so a PR lands the instant required checks go green ([GITHUB.md § 9.4](../../GITHUB.md)), and the integrator role lands green PRs autonomously ([ROSTER.md](../../ROSTER.md)). The human's role degrades to *veto*, not *approve* — the correct load for a strong panel.
- **Weak / contested / irreversible → hold for explicit human accept**, and *put the accept button next to the verdict tally and the diff* so working the gate costs one click, not one terminal. This is H6 done right: not "the dashboard can now merge things," but "the dashboard can work the gate that already exists, and only surfaces the button when the evidence says a human should look."

The reviewer-segment verdict on H6: **a human-in-the-loop approve-gate is worth crossing the read-only boundary for — but only as a gate whose *presence is conditioned on weak or contested evidence.* A gate that fires on every clean panel isn't safety, it's the bottleneck the guide warned about.**

---

## 6. Top 3 needs (ranked by evidence)

Ranked by how often the gap appeared in sampled transcripts and how directly it maps to a reviewer's real, recent pain — past-behavior weight over mock-up reaction ([INTERVIEW_GUIDE.md § 9](../INTERVIEW_GUIDE.md)).

1. **Trust grade + verdict tally, loud, wherever a finding or PR already surfaces (H6-view + findings surfacing, § 2).** Highest frequency, lowest cost, read-only. Every review I do produces a graded finding whose grade currently travels weaker than the finding itself. Make `refuted < unverified < confirmed` (`verify.go:178-180`) a first-class, colored column, and render the panel tally + dissent into the PR via the existing `--with-verdicts` path (`verify.go:204-209`, [GITHUB.md § 9.4](../../GITHUB.md)). No boundary crossing.

2. **Diff annotated by acceptance box — and unbacked boxes flagged (§ 1).** The defect I most often miss is an *absent* one: an acceptance box with no code behind it. A review surface that maps hunks to criteria and shouts the gaps changes my hit rate more than any live feed would. Read-only.

3. **Conditional approve-gate worked from the UI (H6, § 5).** The one write action worth the boundary. Surface an accept/reject button *only when the evidence is weak, contested, or the change is irreversible*; let strong+reversible auto-apply under human veto via `--auto` ([GITHUB.md § 9.4](../../GITHUB.md)). This is the difference between a gate that protects the swarm and one that throttles it.

Explicitly **below the line** for my segment: steer-the-reviewer (H3-steer) — not merely low value but *negative*, because it destroys the independence that makes a verdict worth gating on (§ 3, `verify.go:1-4`).

---

## 7. Un-hypothesized need (RQ5)

The open floor surfaced one thing no hypothesis covers, and it recurred across the failure-sampled transcripts:

**A channel to escalate a *contested* finding without minting a durable false verdict.** Today when a panel splits and I think the majority is wrong, my only moves are: grade with the majority (and let a bad `confirmed` enter every sibling's brief), or file a fresh finding (and start a second, uncoordinated panel). What I want is a **`dacli ask` that attaches to the task and pauses its gate** — a way to raise "this verdict is contested, hold the accept" that a human sees at the gate, without rewriting the log. The escalation primitive exists (`ask` is a first-class command, per [REVIEW.md C1](../../REVIEW.md)); binding it to the approve-gate so a contested review *holds the gate open for a human* rather than resolving silently is the missing edge. A need that recurs across the failures outranks any § 4 hypothesis ([INTERVIEW_GUIDE.md § 8](../INTERVIEW_GUIDE.md)).

---

## 8. Synthesis note

- This is **one source, unverified** — a lead. It confirms nothing on its own; it must recur across ≥3 independent segments before any need here graduates ([INTERVIEW_GUIDE.md § 9](../INTERVIEW_GUIDE.md)).
- **Write-boundary verdict (RQ3):** for the reviewer segment, exactly one hypothesis justifies crossing the read-only projection — **H6, the approve-gate, and only in its evidence-conditioned form.** H3-steer is refused on principle. Everything else (H1, H7, H8) is read-only surfacing that extends today's projection ([`ui/DESIGN.md:20`](../../../internal/features/dashboard/ui/DESIGN.md)) without eroding [DESIGN.md § 2](../../../DESIGN.md).
- Confirmed needs graduate to [PROPOSALS.md](../../PROPOSALS.md) with acceptance tests; a need that contradicts a non-goal goes to the design audit ([REVIEW.md](../../REVIEW.md)), not around it.

_Versioned with the code. Amend via PR when the reviewer role, the verify mechanism, or the gate's mechanics change — a segment writeup run against a stale mechanism is incomparable data._
