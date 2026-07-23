# Walkthrough: one task, end to end

**Status: illustrative.** Nothing here runs yet; this is the spec traced as a single concrete story, using the ledger example threaded through [FORMAT.md](FORMAT.md) and [ARCHITECTURE.md § 6](ARCHITECTURE.md). Writing it is also a test: a step that can't be narrated against the spec is a hole in the spec. (One was found — it's marked.)

Cast: a human; a **root agent** (a Claude Code session — the orchestrator is an agent, never dacli); a spawned **auditor** child, read-only.

---

## 1. The human initializes

```
$ dacli init --name billing --template solo
```

Creates `.dacli/` — `config.yml`, `agents/root.md` (grant `rw`), empty `projects/`, `queues/`, `events/`, and `solo`'s two seed docs. No gates: `solo` is the default because most work should not pay for process ([TEMPLATES.md § 2](TEMPLATES.md)).

## 2. The root agent sets up the project

```
dacli project add "Migrate billing to the new ledger" --slug ledger
dacli task add "Handle the balances properly" --project ledger
```

The second command triggers the lint that pays for the tool:

```
task title: 2 major findings
  1:1  major [vague-words] "Handle" — replace with a specific action verb
  1:26 major [vague-words] "properly" — replace with a defined criterion
```

Three agents given "handle the balances properly" produce three different deliverables. The root agent rewrites:

```
dacli task add "Audit every write path into balances" --project ledger \
  --priority must --estimate 2,5,14 \
  --accept "Every writer of balances is listed with file:line" \
  --accept "Each writer is classified: service-layer or direct"
dacli task add "Add the ledger write shim" --project ledger \
  --priority must --estimate 3,6,15 --depends-on t-…audit:FS \
  --accept "Shim covers the nightly batch path" \
  --accept "Reconciliation suite green"
dacli risk add "Nightly batch may bypass the service layer" \
  --impact high --likelihood medium \
  --indicator "reconciliation diffs appearing only after 02:00 UTC" \
  --action "audit the batch write path before building the shim"
dacli note add decision "Ledger writes stay synchronous" --project ledger \
  --rejected "async queue + eventual reconciliation" \
  --because "reconciliation cost exceeds the ~40ms win at current volume"
```

On disk: two files in `tasks/open/`, one in `risks/` (rank 2, has its required action), one in `notes/decisions/`. Estimates are three-point or refused — the pessimistic number is where the unexamined risk lives.

```
$ dacli next
1. 001-audit-write-paths   must · zero slack · Te 6.0 (elicitation → 3–12)
```

CPM says the audit gates everything; MoSCoW agrees; risk-value agrees (it's also the task that can invalidate the plan). One recommendation, three frameworks concurring.

## 3. Spawn the auditor

```
$ dacli spawn --role auditor --task t-…audit --budget 8000
```

One call: WIP check → child identity minted at `ro` (role ceiling ∧ parent grant — attenuation wins) → runtime launched with its **sandbox flags set to read-only**, which is where enforcement stops being cooperative ([RUNTIMES.md § 8](RUNTIMES.md)) → brief assembled and delivered. The brief is, almost verbatim, the worked example in ARCHITECTURE § 6: acceptance, goal chain, out-of-scope, the sync-writes decision (so the child cannot re-propose the async queue), the rank-2 risk with its 02:00 indicator, glossary, shortcut catalog.

## 4. The child works — and everything comes back as events

The auditor greps, reads, and finds it: the nightly batch job writes `balances` directly. It reports *the moment it learns the thing*:

```
DACLI_AGENT=$TOKEN dacli note add finding \
  "cron/settle_batch.go:112 writes balances directly, bypassing the service layer. \
   Any shim wrapping only the service layer will miss it." \
  --about t-…audit --severity major
```

A read-only agent writing? Yes — this lands as `events/2026/07/21/01J8…-a01J8…-finding.md`, a **new file**, which is all an `ro` grant permits and all reporting requires. No lock, no contention: a sibling writing in the same instant creates a different ULID.

The child checks its two acceptance boxes' evidence into the finding, proposes completion (`propose-status` event), and exits inside budget. Suppose it hadn't — killed at 8,000 tokens, the finding file already exists. **Partial failure keeps the partial work**; that is why results travel through the workspace and never through stdout parsing.

## 5. The parent evaluates against fixed criteria

```
$ dacli events tail        # finding visible immediately — reads fold in pending events
$ dacli sync               # owner materializes: finding → note + task ## Log;
                           # propose-status → git mv tasks/open/… tasks/done/…
```

The supervision loop terminates here not because the parent is satisfied — because the acceptance boxes, written before the child existed, are checked. That external criterion is the entire difference between this and the agent chat this design refuses to build ([RUNTIMES.md § 7](RUNTIMES.md)).

## 6. The shim task, and a refusal that is an answer

The root agent claims the shim task itself, builds against the now-recorded batch-path constraint, runs `dacli run test` (a `run` event; `uses` will be recomputed at sync), and tries to finish early:

```
$ dacli task done t-…shim
refused (exit 3): acceptance unmet — "Reconciliation suite green" unchecked
                  definition of done — shortcut `test` has no passing run event
```

Exit 3, not 1: *no* is information. The agent's correct move is to fix or `ask` — never to retry, which is precisely why refusal and failure are different numbers. Suite passes, boxes check, `task done` succeeds, folder moves.

> **Spec hole found while writing this step** (the walkthrough doing its job): `--accept` and `--estimate` flags on `task add`, and `--indicator`/`--action` on `risk add`, appear in no command spec — the tables list commands but not their flag surfaces. Recorded as REVIEW G13; the flags used here are the proposal.

## 7. Closure

```
dacli retro t-…shim        # went well / didn't / improve → durable note
dacli github sync --dry-run
dacli github sync          # issues #12, #13 created, marker comments embedded,
                           # both closed with status mirrored; finding lands as
                           # an issue comment, attributed
```

The human, who touched nothing since § 1, reads the whole story on GitHub. In Obsidian, the same story is the vault graph: task ↔ decision ↔ finding ↔ risk, already linked.

## 8. The scorecard

Every step exercised an invariant; that mapping is the point of the tool:

| Step | Invariant at work |
|---|---|
| 2 | Ambiguity lint before work, not after ([SPM.md](SPM.md)) |
| 2 | Estimates are ranges; scalar refused |
| 3 | Attenuation ∧ role ceiling; sandbox = real enforcement when spawned |
| 4 | `ro` agents report via append-only events; ULID names can't collide |
| 4 | Partial work survives a dead child |
| 5 | Reads fold pending events; only the owner materializes |
| 6 | Refusal (3) ≠ failure (1); DoD enforced at `done` |
| 7 | GitHub is a projection; humans enter as events |

## 9. Zooming out: the perpetual loop

Everything above is one task, spawned by hand. `dacli loop` runs that same shape — spawn → wait → land — as a **governed, repeating cycle**, so a maintenance team runs without a human re-triggering it every time it empties its backlog. Unlike §§1–8, this section is not illustrative: `internal/features/orchestration` is implemented and tested (`governor_test.go`, `driver_test.go`, `state_test.go`).

```bash
dacli loop --project ledger --width 3 --max-cycles 5        # bounded: 5 sprints, then stop
dacli loop --project ledger --window-tokens 2000000 --yolo  # perpetual, budget-governed
```

### The sprint model: one cycle, six phases

Each cycle walks the phases a real team walks each sprint, then goes around again (`runCycle` in `internal/features/orchestration/orchestration.go`):

| Phase | What actually runs |
|---|---|
| **Plan** | `readyTasks` — the open backlog whose finish-relation dependencies are all done, capped to `--width` |
| **Implement** | one `dacli spawn --task <ref> --role <impl-role> --detach --worktree [--pr]` per task in the batch |
| **Test** | `dacli wait` blocks until the whole detached wave finishes and finalizes its outcome |
| **Land** | see below — the default (`--pr`) and local (`--no-pr`) models differ here |
| **Review** | a reviewer is spawned against a standing "Continuous improvement" task whose charter is to *file* the next evidence-based improvement — never to implement it |
| **Retro** | `dacli retro --project <slug>` harvests the cycle for the record |

The review phase is the engine: it regenerates the backlog, which is why the loop is self-feeding instead of stalling the moment the initial backlog empties.

### The governor: a pure decision engine

No cycle runs because "keep going" is the default — every checkpoint passes through the `Governor` (`internal/features/orchestration/governor.go`), a decision function with no side effects (it never spawns, sleeps, or touches the network), which is what makes the perpetual machine testable without burning a token:

| Decision | Trigger | Knob |
|---|---|---|
| `Idle` | Backlog is empty | never invents work — sleeps `--idle` and re-scans |
| `SleepWindow` | Rolling token budget is spent | `--window-tokens N --budget-window DUR` |
| `Halt` (bound) | `--max-cycles` reached | operator-set bound |
| `Halt` (thrash guard) | N consecutive cycles land nothing on trunk | `--no-progress-halt` (default 3) |
| `Halt` (kill switch) | `.dacli/STOP` exists | `touch .dacli/STOP` to stop; remove it to resume |

Progress is measured by **trunk actually advancing** — commits that reached `main`, local or `origin` — never a task-status delta. Under the default `--pr --auto` landing model, GitHub merges each PR asynchronously once its own CI passes, so a task the loop closes this cycle may merge a cycle or two later, or never; a late merge resets the thrash streak, and only trunk that never moves across `--no-progress-halt` consecutive cycles halts the loop.

An unbounded run with no stop condition is refused outright: set `--max-cycles`, keep the thrash guard on, or pass `--yolo` to explicitly accept a genuinely perpetual run. `dacli loop status --project <slug>` reads the last persisted checkpoint (cycle, trunk marker, tokens spent this window, ready backlog) without waiting on a running loop; `dacli loop --dry-run` previews one cycle's commands with nothing actually spawned; `dacli loop --project <slug> --width N --advise` reports the expected per-sprint token cost band at that width — `width × median tokens/run for --impl-role` plus one review spawn's median for `--review-role`, from `dacli calibrate`'s measured bands (grouped by role alone, since the loop does not pin a model/runtime ahead of a spawn) — and, like `spawn --advise`, changes nothing and needs no stop condition.

### Landing: auto-merge, and the integrator role

Two mechanisms keep "a broken main never happens" true with no human watching:

1. **Inline, per-PR auto-merge.** Every `--pr` implementer runs `dacli pr --task <ref> --with-verdicts --auto` as the last step of its own git workflow — the exact brief text `dacli` hands every read-write child (see [PROMPTS.md](PROMPTS.md)). `--auto` queues GitHub's *native* auto-merge (`gh pr merge --auto --merge`): the PR lands itself the instant its required checks go green, and degrades to "left open for a human" when the repo has no branch protection — never a silent local merge over red or pending CI.
2. **The `integrator` role.** A standing, spawnable `rw` reviewer-kind agent (see its row in [ROSTER.md](ROSTER.md)) whose entire charter is release management: sweep open PRs on done tasks, merge the ones with green `gh pr checks`, queue `--auto` on ones still running, and refuse to merge red CI — filing a finding naming the failing check instead. It never implements. Spawn it as a standing backstop wherever a PR might need landing outside the loop's own inline path — a human-triggered spawn wave, a PR opened without `--auto`, a stuck merge — the same merge discipline the loop applies to itself, callable on demand.
