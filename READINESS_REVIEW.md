# dacli Readiness Review — trajectory-as-product

**Date:** 2026-07-27 · **Commit:** `18cf251` · **Method:** 4 parallel audits + empirical measurement of dacli's own repo (which dacli built, so its history is a live sample of the output).

**The job being assessed:** generate many complete software repositories at volume, where each repo's *development trajectory* — commits, PRs, review, task records, event log — is the product.

---

## Verdict

**Not ready.** Not because of missing features, but because of one structural fact:

> **dacli's "done" is an unverified assertion.** `store.CheckAllAcceptance` flips every acceptance checkbox to `[x]` without ever reading the criterion, and there is **no test or coverage gate anywhere in the codebase**.

For a normal tool that is a quality problem. For a tool whose *record* is the product, it is fatal: every generated repo ships labels that claim work was verified when nothing was checked. That is label noise injected at the source.

Three blocking classes, in priority order:

| | Class | One-line statement | Blocks |
|---|---|---|---|
| **A** | **Truth** | The trajectory can claim work that wasn't done or verified | Data validity |
| **B** | **Greenfield** | dacli maintains existing repos; it cannot build new applications | Producing repos at all |
| **C** | **Signal & fingerprint** | 58% of generated history is the loop narrating itself; the rest is trivially clusterable | Buyer-facing quality |

The good news, stated up front: **volume mechanics are genuinely solid.** N independent loops over N separate repos is safe — no global locks, no shared paths, no ULID collision risk, no port/socket contention. Scaling out is not the problem.

---

## A. Truth — the label-integrity blockers

The `--pr` loop path was hardened in the last audit (`branchHasWork`, `prLandStatus`). These holes are elsewhere and are worse, because they sit on the *acceptance* path rather than the *landing* path.

**A1 — Acceptance boxes are checked without reading the criteria. CRITICAL. CONFIRMED.**
`store/store.go` `CheckAllAcceptance` unconditionally sets every `- [ ]` to `- [x]`. `--verify` is optional; on the hardened merge path `orchestration.go:601` calls `accept NNN --force` with **no `--verify` at all**. A merged PR proves commits landed; it does not prove *"returns 404 on missing user"* is true.

**A2 — One verify command closes N unrelated tasks. CRITICAL. CONFIRMED.**
`acceptance.go:157-162`: `acceptAll` runs the verify command **once**, then closes every proposed task. `ship.go:110` always calls `accept --all --force`. So a single `go build ./...` closes tasks 007, 012, and 019 — including ones whose work is absent.

**A3 — Self-certification. CRITICAL. CONFIRMED.**
`task check --all` lets the owner tick its own boxes; `task done` "verifies" only that boxes are ticked. The agent that did the work certifies the work, with no external referent.

**A4 — No test gate exists, and the gate system is architecturally closed to one. CRITICAL. CONFIRMED.**
`internal/gates` predicates are `project_sections | glossary | decisions | tasks | risks | retro` — all markdown inspection. Gates never touch git, build, or tests, and the design note explicitly forecloses a scripted predicate. `musts_done` consumes the unverified close from A1. **A repo can be generated where tests fail or don't exist, coverage is nil, and every record reads `done` with all boxes checked, a merged PR, a filled retro, and a passing stage gate.**

**A5 — Commit attribution is an env var. HIGH. CONFIRMED.**
`vcs.go:137-149` sets `--author` from `DACLI_AGENT`. Any process holding a child's token commits *as* that child — history can attribute work to an agent that never ran, undetectably.

**A6 — `ship` closes tasks before integrating. HIGH.** A later conflict aborts, but the tasks are already `done` on disk and are never reopened.

**A7 — Verify panel records "refuted" when the panel never ran. MEDIUM.** If every runtime crashes, `verify.go:181-190` stamps a refutation nobody made.

**A8 — Calibration/velocity derive from unverified closes. MEDIUM.** (Token/burn figures read real `usage.txt` and are honest.)

### What A requires
1. **Per-task verification, not batch.** One verify command per task, its exit code recorded as evidence on that task.
2. **A real test gate.** Add file-existence and command-exit predicates to `internal/gates` (this contradicts a stated design principle — that principle should change for this use case), plus coverage capture.
3. **Evidence-bound acceptance.** A box may only be checked with a recorded artifact: a passing command, a diff hunk, a test name. `CheckAllAcceptance` should not exist in its current form.
4. **Independent certification.** The reviewer role, not the implementer, closes the task.

---

## B. Greenfield — dacli cannot currently build a new application

**B1 — There is no greenfield path.** The only onboarding verb is `adopt`, explicitly an existing-repo verb. On an empty directory: `codebase map written (0 files, 0 languages)`. Spec → backlog is entirely manual. No `dacli new`, no spec ingest, no scaffold.

**B2 — The `product` template deadlocks the loop. CONFIRMED empirically.**
`internal/features/orchestration` imports `internal/gates` **nowhere** — the loop is stage-blind. Meanwhile `phaseGate` (`execution.go:852`) refuses an implementer spawn during a non-implementation phase:

```
dacli: project recipe-app is in the discovery phase; a implementer role has no work
here (allowed: researcher, reviewer). Advance the stage first…
```

So `product` + `loop` = **every BUILD spawn refused forever**, until the thrash guard kills it. The one template shaped for building a product is the one that cannot run.

**B3 — The review anchor is written for an existing codebase.** It instructs: *"Survey the code, tests, CI… grounded in evidence… do NOT invent speculative work."* On an empty repo there is no evidence, so nothing is filed. Confirmed live:

```
● cycle 1: backlog empty — no evidence-based work; idling rather than inventing work
```

A greenfield repo with a stated goal idles forever. Needs a **spec-decomposition anchor**, licensed to invent work from the Goal.

**B4 — No planner, no automatic decomposition.** `wbs` renders a tree someone else authored; `critical-path` analyses a DAG someone else built. The `product` template allows a `planner` kind but no roster ever creates one. A human must author the WBS.

**B5 — Go is hardcoded on the default path.** `implRole` defaults to `fixer`, `reviewRole` to `go-auditor` — confirmed live on a *recipe web app*. `git_workflow.md` orders every child to `gofmt -w` its files regardless of stack.

**B6 — The project object has nowhere to put a spec or architecture.** Only Goal / Constraints / Out of scope / Success criteria.

**B7 — LAND/REVIEW assume GitHub from cycle one**, but a greenfield repo has no origin, no PRs, no CI.

**B8 — No release arc.** Nothing versions, tags, writes a CHANGELOG, or declares v1 done. The loop's only terminal condition is an empty backlog.

---

## C. Signal & fingerprint — measured on dacli's own repo

| Signal | Measurement |
|---|---|
| Bookkeeping/merge commits | **251 of 429 (58%)** |
| Commits touching *only* `.dacli/` | **61 of last 200 (30%)** |
| Most common commit message | `"ship: record workspace after integrating 0 task(s)"` — **verbatim ×61** |
| Timeline | 429 commits / **6 days**, all hours |
| Tracked files | 141 Go (34.8k LOC) vs **1,135 `.dacli/` files** |
| PR reviews across 30 merged PRs | **0 reviews, 0 line comments, 0 issue comments** |
| PR bodies | all open `"Implements dacli task NNN-slug."` |

**C1 — The majority of generated history is the loop narrating itself.** A commit message repeated identically 61 times, saying it integrated *zero* tasks, is the opposite of the history quality a buyer pays for. **Fix: keep `.dacli` out of the product repo's history** (separate branch/orphan ref/artifact), and vary commit-message generation.

**C2 — The review data never reaches the artifact the buyer inspects.** Findings and verdicts live in `.dacli`; GitHub PRs carry no review. `pr --with-verdicts` posts one whole-PR comment and is only *suggested* in a prompt template.

**C3 — The structural spine is hardcoded.** Varyable today: prompts, stage templates, roster (best axis). Hardcoded: `.dacli` dir name, `Dacli-*` commit trailers, `@agent.dacli` author domain, branch scheme `dacli/%03d-%s`, `<!-- dacli:… -->` HTML markers in every issue body, the per-cycle retro sentence, task-log verbs. **A trivial classifier hits ~100%.**

**C4 — CI history is hollow.** `pr checks` is consumed as a boolean and **"no checks reported" counts as pass** (`lifecycle.go:960`), so a repo with zero CI merges everything green and carries no check history. Nothing in dacli authors workflow files.

---

## GitHub integration roadmap

Ranked by impact on trajectory value. dacli makes **zero `gh api`/GraphQL calls** today — everything is porcelain, which caps achievable richness.

| # | Integration | Implementation | Why it matters |
|---|---|---|---|
| 1 | **Line-level review comments** | `gh api .../pulls/{n}/reviews -f event=COMMENT -F 'comments[][path]=…' -F 'comments[][line]=…'` | The single highest-value artifact. Findings already carry file refs. |
| 2 | **Review states + back-and-forth** | `APPROVE` / `REQUEST_CHANGES`, then re-review after the fixer pushes | A repo where no PR was ever blocked reads as synthetic |
| 3 | **Author real CI workflows** | Emit `.github/workflows/*.yml`, language-detected | Prerequisite for genuine check history; buyer explicitly wants test coverage |
| 4 | **Real check history** | `gh run list --json conclusion,workflowName`, `gh api .../check-runs`; record into eventlog; fix the no-checks⇒pass escape | Failing→passing history is high-value trajectory |
| 5 | **Coverage as a published artifact** | `go test -coverprofile` (per-language equivalent), publish to CI + PR comment | Directly serves the buyer's stated requirement; currently has *no* GitHub expression |
| 6 | **Draft → ready transitions** | `pr create --draft`, later `gh pr ready` | Cheap, high realism |
| 7 | **Releases + tags + notes** | `gh release create vX --generate-notes` | `EventRelease` and a `release` phase already exist as concepts |
| 8 | **Milestones** | `gh api .../milestones` + `issue edit --milestone` | Projects map 1:1 |
| 9 | **Issue↔PR reverse linking** | Comment on the issue when its PR opens | Only `Fixes #N` exists today |
| 10 | **CODEOWNERS** | Emit from the role roster's path scopes (already declared) | Produces authentic reviewer-assignment events |

---

## Roadmap

**Phase 0 — Truth (blocks everything).** Per-task verify; command-exit + file-existence gate predicates; coverage capture; evidence-bound acceptance; reviewer-certifies-not-implementer; fix A5–A8.

**Phase 1 — Greenfield.** `dacli new` from a spec; planner role + spec→DAG decomposition; make the loop stage-aware (read `gates.Status`, auto-advance, pick phase-appropriate roles); bootstrap cycle (repo/remote/CI before PR-first); stack detection replacing the Go defaults; Spec/Architecture/Stack sections on the project.

**Phase 2 — Output quality.** `.dacli` out of product history; commit-message variation; line-level reviews + review states; authored CI workflows + real check history; draft→ready; releases.

**Phase 3 — Diversity.** Make the hardcoded spine configurable: workspace dir name, trailer scheme, author domain, branch naming, GH markers, retro/log phrasing. Per-repo variation profiles.

**Phase 4 — Volume hardening.** `runs prune` deletes live runs' state (high); `CreateTask` seq TOCTOU; non-atomic `Governor` state write; `eventlog.Sync` double-apply; a real `--resume`.

---

## The honest bottom line

Phase 0 is not optional and is not cosmetic. Until acceptance is evidence-bound and a test gate exists, **every repo dacli generates ships a trajectory whose completion labels are unverified assertions** — the exact defect class that makes training data actively harmful rather than merely mediocre. Phase 1 is what makes generation possible at all; Phases 2–3 are what make the output worth buying.
