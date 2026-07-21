# Full design review — 2026-07-21

Adversarial self-audit of the complete spec (8 docs, ~1,800 lines; Go skeleton, ~2,500 lines) after four design sessions. Severities use the review ladder from the SPM layer: **major** = fix not obvious, needs work; **moderate** = fix clear, needs review; **minor** = fix obvious.

Method note: every document under review was written by the same author now reviewing them, in the same week. The consistency defects below are what that method catches; what it structurally cannot catch is a shared blind spot, which is an argument for the multi-runtime verification panel applied to this repo's own docs, someday.

Companion piece: the structural verdict is normative in [ARCHITECTURE.md](ARCHITECTURE.md), not repeated here.

---

## 1. Consistency defects — found and fixed in this pass

| # | Sev | Defect | Fix |
|---|---|---|---|
| C1 | **major** | `help ask` / `help answer` / `help escalate` were **unreachable**: `Main` intercepts `args[0] == "help"` for usage before dispatch runs. An entire feature namespace, dead on arrival, and the duplicate-path test couldn't see it. | Renamed to `ask` / `answer` / `escalate`; added a reserved-word reachability test. |
| C2 | **major** | Queues carried mutable state (`cursor`) with **no owner** — the one object anybody could rewrite, silently violating the single-writer invariant the whole concurrency design rests on. | Queues now have `owner:`; only the owner advances. A queue is a checklist walked by one agent, not a work-distribution mechanism. |
| C3 | **major** | Shortcut `uses:` was an incremented-in-place counter, which would make the shortcut file the one object every agent writes concurrently — the exact contention the event log exists to prevent. | `uses` is derived: recomputed from `run` events at sync, read-only cache otherwise. |
| C4 | moderate | FORMAT.md's event-kind table was missing `help`, `answer`, and `run` — three kinds added to the model but never to the format spec that claims to be the stable interface. | Table updated. |
| C5 | moderate | Two unreconciled stage vocabularies: project `stage:` uses cone names (`definition`…`design`), template stages use process names (`inception`…`transition`), and TEMPLATES.md claimed they "map onto" each other without saying how. An implementer would have guessed. | Every template stage now declares an explicit `cone:`. |
| C6 | moderate | "Five object types" claims in DESIGN.md § 4 and the `model` package doc, three layers of satellites later. | Reworded: five collaboration-core types plus named satellites. |
| C7 | minor | Stale `RUNTIMES.md § 14/§ 17` cross-references after section insertion. | Renumbered. |

The pattern behind C1–C3 is worth naming: all three are **invariants enforced nowhere**. The single-writer rule and the command-dispatch rule lived in prose, so violating them compiled cleanly and passed tests. Where an invariant can be a test (C1's reserved-word check) it now is; the ownership rules become enforceable only when the spine exists, which is one more argument for building it first.

## 2. Gaps — open, with proposed resolutions

| # | Sev | Gap | Proposed resolution |
|---|---|---|---|
| G1 | **major** | **No MCP spec.** "Two front ends, one core" since day one, agents are the primary audience, and the agent-preferred surface is the only one without a document. | `docs/MCP.md` next, before any L5 implementation. Contract sketch in ARCHITECTURE § 4: one tool per command, same JSON, generated from the same table. |
| G2 | **major** | **No worked example of a brief existed anywhere** — 1,800 lines of spec for a tool whose entire output is one artifact, and the artifact was never shown. | Fixed in this pass: ARCHITECTURE § 6 is now the canonical example, including attributed-quote rendering of third-party content. |
| G3 | moderate | No exit-code or machine-output contract; agents would have parsed stderr. | Fixed in this pass: ARCHITECTURE § 4. The refusal/failure distinction (3 vs 1) is the load-bearing part. |
| G4 | moderate | Task-ID collision under concurrent creation — open since the first session. | **Resolved**: ULID is the true id (collision-free at creation); `NNN` is a filename alias assigned by the single materializer (the owner) at sync. FORMAT.md updated, DESIGN open-question 5 closed. |
| G5 | moderate | No lifecycle: nothing closes a project, archives done tasks, or bounds workspace growth beyond event compaction ("sketched, not designed"). | Propose `dacli archive <project>`: moves to `projects/_archive/`, compacts its events, excludes it from `status`/`next`. Needs a spec pass. |
| G6 | moderate | Same-identity multi-process races: two shells acting as one agent can interleave writes to that agent's own objects. Atomic rename means last-write-wins, not corruption — but it was stated nowhere. | Stated in ARCHITECTURE § 5 as honest scope. Optional advisory flock later; not v0.1. |
| G7 | moderate | Ambiguity-linter noise: qualifiers/positional/temporal at moderate fire on ordinary prose ("move the file **after** tests pass"). A linter that cries wolf trains agents to skip it — the exact failure it exists to prevent. | **Resolved**: asymmetric scope policy in SPM.md — titles + acceptance at moderate, bodies at major-only, `--strict` widens. |
| G8 | minor | `Route()` dead-ends when a role *owns* the path but no `escalate_to` edge reaches it — a config gap indistinguishable from "no owner." | **Resolved in spec** (TEAM.md): the error names the owning role and the missing edge. |
| G9 | minor | `dacli next` behavior when tasks lack estimates (CPM needs durations) was unspecified. | Degrade to MoSCoW-then-dependency order, announced per axiom 6. One line, needs to land in SPM.md. |
| G10 | minor | Panel verdicts (`verify --require 2`) have no recorded form. | A `verdict` field on finding events; counts derived. Spec with runtimes. |
| G11 | minor | Config layering (workspace / `~/.dacli` / env) and format migration beyond "additive after 1" unspecified. | Defer both until v0.2 forces them; note here so they're not forgotten. |
| G12 | minor | No test story for runtime adapters without paid API calls. | Ship a `mock` runtime — `generic-exec` pointed at a fixture script. Cheap, and CI needs it the day L5 starts. |

## 3. Feature proposals from the review

Ranked; none are commitments.

1. **`dacli context --explain`** — show why each item made the brief and what was cut at what cost. The brief is the product; when it's wrong, this is the only debugger. (Small, pure-L4.)
2. **`dacli task done` enforces the definition of done** — the template DoD currently binds nothing; done-time is where it has teeth.
3. **Attributed-quote fencing of third-party content in briefs** — now in the ARCHITECTURE example; makes injection visible, not impossible. Cheapest available mitigation for the design's worst open problem.
4. **Brief caching by content hash** — briefs and gate checks re-derive identical output; a hash key makes panels and repeated spawns near-free. (Optimization; after dogfooding proves the shape.)
5. **`dacli template diff`** — vendored templates drift silently from their source; promote from TEMPLATES open-questions to planned.
6. **Unified `dacli doctor`** — workspace, runtime, and github doctors under one health surface with one exit-code convention.

## 4. What the review did *not* find

Stated so the clean bill is explicit, not an oversight: the event-log concurrency model survived adversarial reading intact once C2/C3 were fixed — every mutation is either owner-writes-own-file or a new ULID-named file. The permission story's honesty (cooperative → enforced-when-spawned) held up. The three-surfaces framing (Obsidian/GitHub/dacli over one store) generated no contradictions anywhere in the eight docs. And the "runs agents, not work" boundary, restated once, absorbed both later feature waves (slash commands, templates) without further erosion.

The worst open problem remains what it was: **cross-tree prompt injection** ([RUNTIMES.md § 18](RUNTIMES.md)). Nothing in this review shrank it; proposal 3 above only makes it auditable.

---

## 5. Dispositions — upgrade pass, later the same day

The register above is append-only; what changed since it was written:

- **G1 resolved** — [MCP.md](MCP.md) written. It *overturned* ARCHITECTURE § 4's one-tool-per-command promise (a 50-schema catalog is the per-agent tax this design refuses elsewhere) in favor of a tiered surface: fourteen core tools + a `cli` escape hatch, refusals as results so clients never retry a policy "no", identity bound at server launch so tokens stay out of transcripts. ARCHITECTURE § 4 amended to match, with the correction owned in place.
- **G7 was already landed** in SPM.md at review time; **G9** (estimate-less `next` degrades to MoSCoW-then-dependency, announced), **G10** (panel verdicts recorded as `verdict:`-bearing finding events, tallies derived — same rule as shortcut `uses`), and **G12** (the `mock` fixture-script adapter as the entire L5 CI story) are now landed in SPM.md and RUNTIMES.md.
- **Proposal 2 landed** — the DoD binds at `dacli task done`, refusal names the unmet criterion (TEMPLATES.md).
- **New: G13 (moderate)** — found *by writing* [WALKTHROUGH.md](WALKTHROUGH.md), which is exactly what it was for: no command spec anywhere defines flag surfaces (`task add --accept/--estimate`, `risk add --indicator/--action`, …). The command tables list verbs, not signatures; the walkthrough's flags are the proposal. Resolve when the CLI grows real arg parsing — the `--json` output shapes and these flag names should be specified together, as one contract.
- **Also added**: [docs/README.md](README.md) index with per-doc status labels; status headers on SPM/TEAM/SHORTCUTS distinguishing implemented engines from stubbed commands.

Still open after this pass: G5 (lifecycle/archive), G6 (same-identity races — scoped honestly in ARCHITECTURE § 5), G11 (config layering, deferred until v0.2 forces it), G13 (above), and the injection problem, unchanged.
