# Walkthrough: one task, end to end

## The default agent journey

The primary path is intentionally smaller than the complete command catalog:

```text
inspect → plan → claim → implement → verify → review → PR → CI → merge
```

Start with `dacli start --profile inspect` when an agent only needs to audit,
or `--profile task` for one bounded change. Neither mode requires estimates or
portfolio configuration. Estimates become important for `wave`, `loop`, and
`service`, where they drive capacity, critical-path slack, worker timeouts, and
spend projections. `dacli start --dry-run` says this in its resolved plan.

`dacli init --roster agents` seeds six provider-neutral capabilities: planner,
implementer, security implementer, reviewer, security reviewer, and integration
owner. The preset intentionally records no runtime or model. Bind its roles to
the coding harness already selected for the project; a second harness appears
only under an explicit hybrid profile.

The executable clean-fixture version is
[`docs/examples/default-agent-workflow.sh`](examples/default-agent-workflow.sh):

```bash
go build -o /tmp/dacli ./cmd/dacli
DACLI_BIN=/tmp/dacli docs/examples/default-agent-workflow.sh
```

It uses local integration so it runs without credentials. For the ordinary
GitHub boundary, replace its final local integration with these observable
steps; the integration owner, not the worker, owns them:

```bash
dacli push --task 001
dacli pr --task 001 --base main
dacli pr wait --task 001 --timeout 1800 --json
dacli review projection --task 001 --json
dacli pr land --task 001 --base main --merge
dacli branches audit --project ledger --json
# Apply only the exact returned plan id:
dacli branches prune --project ledger --apply-safe <plan-id>
```

The compact forms delegate to the canonical engines: `pr wait` consumes `pr
diagnose`, `pr land` consumes PR integration, and `branches audit/prune`
consume the content-addressed cleanup planner. `route <path>` likewise aliases
`team route`, while `status --project` and `next --critical-path` keep project
selection explicit for an agent. The complete catalog and MCP executor expose
the same command metadata through `capabilities --json`.

When the selected profile requires independent delivery review, the loop adds
a gate between verification and landing. It launches the configured reviewer
read-only with `--review --structured-review-result`; the reviewer returns a
bounded `DACLI_REVIEW_RESULT` envelope, and the parent validates reviewer
identity, runtime/model/grant, branch commit, and exact tree before recording
`independent-review-result/v1`. An approval applies only to that tree. Requested
changes get at most the configured correction turns and require a fresh review
of the corrected tree; missing, stale, inconclusive, or infrastructure-only
results fail closed. Before launch, the parent proves that it can publish the
structured result. If that channel disappears after analysis, the run becomes
`handoff-required` with the validated finding IDs, failed operation, and safe
recovery action; the owner restores the channel and reruns the exact-tree
review without reading the raw transcript or inferring approval from silence.
`dacli review projection --task <ref> --json` exposes only
the public-safe verdict and line-comment projection, not private evidence or
agent/runtime identities.

Lifecycle ownership is singular: `task done` closes work performed by its
owner; a spawned worker proposes and the workspace owner uses `accept` to
validate and close it; the integration owner uses `integrate` to land accepted
branches; the wave owner uses `ship` when accept, integration, durable record,
and optional publication must be one governed wave tail. Run `dacli task
--help` for a family, a leaf command with `--help` for its flags, and `dacli
help --all` for advanced/recovery tools.

**Status: illustrative.** The commands here run; this traces the workflow as a single concrete story, using the ledger example threaded through [FORMAT.md](FORMAT.md) and [ARCHITECTURE.md § 6](ARCHITECTURE.md). Writing it is also a test: a step that can't be narrated against the tool is a hole in the tool.

Cast: a human; a **root agent** (running in any configured coding-agent CLI — the orchestrator is an agent, never dacli); a spawned **auditor** child, read-only.

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

One call: WIP check → child identity minted at `ro` (role ceiling ∧ parent grant — attenuation wins) → runtime launched with its declared read-only sandbox arguments, but only after `runtime doctor` has verified that exact local runtime and sandbox declaration ([RUNTIMES.md § 8](RUNTIMES.md)) → brief assembled and delivered. An unknown, stale, or failed probe is refused (exit 3); `--cooperative` is the explicit, loudly announced escape hatch. The brief is, almost verbatim, the worked example in ARCHITECTURE § 6: acceptance, goal chain, out-of-scope, the sync-writes decision (so the child cannot re-propose the async queue), the rank-2 risk with its 02:00 indicator, glossary, shortcut catalog.

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
dacli github sync ledger --dry-run
dacli github sync ledger   # issues #12, #13 created, marker comments embedded,
                           # both closed with status mirrored; internal finding
                           # stays withheld unless separate authority + request exist
```

The human, who touched nothing since § 1, reads the whole story on GitHub. In Obsidian, the same story is the vault graph: task ↔ decision ↔ finding ↔ risk, already linked.

## 8. The scorecard

Every step exercised an invariant; that mapping is the point of the tool:

| Step | Invariant at work |
|---|---|
| 2 | Ambiguity lint before work, not after ([SPM.md](SPM.md)) |
| 2 | Estimates are ranges; scalar refused |
| 3 | Attenuation ∧ role ceiling; verified local read-only sandbox or refusal |
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
| **Delivery review** | when required by the profile, an independent read-only reviewer returns a structured, identity/tree-bound verdict; corrections and re-review are bounded |
| **Land** | see below — the default (`--pr`) and local (`--no-pr`) models differ here |
| **Review** | a reviewer is spawned against a standing "Continuous improvement" task whose charter is to *file* the next evidence-based improvement — never to implement it |
| **Retro** | `dacli retro --project <slug>` harvests the cycle for the record |

The periodic review phase is the improvement engine: it regenerates the
backlog, which is why the loop can find the next evidence-backed improvement
instead of stalling when the initial backlog empties. It is distinct from the
per-task delivery-review gate above; one discovers future work, while the
other decides whether an exact implementation tree may proceed to landing.

### The governor: a pure decision engine

No cycle runs because "keep going" is the default — every checkpoint passes through the `Governor` (`internal/features/orchestration/governor.go`), a decision function with no side effects (it never spawns, sleeps, or touches the network), which is what makes the perpetual machine testable without burning a token:

| Decision | Trigger | Knob |
|---|---|---|
| `Idle` | Backlog is empty | never invents work — sleeps `--idle` and re-scans |
| `SleepWindow` | Rolling token budget is spent | `--window-tokens N --budget-window DUR` |
| `Halt` (bound) | `--max-cycles` reached | operator-set bound |
| `Halt` (thrash guard) | N consecutive cycles land nothing on trunk | `--no-progress-halt` (default 3) |
| `Halt` (kill switch) | `.dacli/STOP` exists | `touch .dacli/STOP` to stop; remove it to resume |

Progress is measured by **trunk actually advancing** — commits that reached `main`, local or `origin` — never a task-status delta. Under an explicitly configured `auto_merge=true` PR landing model, GitHub merges each PR asynchronously once its own CI passes; the safe profile default leaves the PR open after the controller pushes and creates/reuses it. A late merge resets the thrash streak, and only trunk that never moves across `--no-progress-halt` consecutive cycles halts the loop.

An unbounded run with no stop condition is refused outright: set `--max-cycles`, keep the thrash guard on, or pass `--yolo` to explicitly accept a genuinely perpetual run. `dacli loop status --project <slug>` reads the last persisted checkpoint (cycle, trunk marker, tokens spent this window, ready backlog) without waiting on a running loop; `dacli loop --dry-run` previews one cycle's commands with nothing actually spawned; `dacli loop --project <slug> --width N --advise` reports the expected per-sprint token cost band at that width — `width × median tokens/run for --impl-role` plus one review spawn's median for `--review-role`, from `dacli calibrate`'s measured bands (grouped by role alone, since the loop does not pin a model/runtime ahead of a spawn) — and, like `spawn --advise`, changes nothing and needs no stop condition.

Loop workers receive five minutes of wall-clock time per expected estimate point (PERT Te), with a five-minute floor for unestimated or sub-point work. Thus work above Te 1 is not silently killed at the spawn command's historical 300-second default. Pass `--worker-timeout SEC` to override that derived policy for every implementation and review worker launched by the loop.

Before the first implementation worker, the loop writes one versioned cycle
preflight to `.dacli/loop/<project>-preflight.json`. It records the resolved
task/role/runtime/model/grant/claims, implementation and reviewer WIP, worker
timeouts, STOP and rolling/cycle budgets, landing/check/GitHub observability,
and every verification command with its working directory. A policy or
capability mismatch is `permanent_refusal` (exit 3); an external observation
failure is `transient_failure` (exit 1) and names the bounded retry. An explicit
implementation role that is too small still refuses. The workspace owner can
accept that risk only with both `--capacity-override-reason TEXT` and a future
`--capacity-override-expires RFC3339`; the preflight durably records the task,
role, capacity delta, actor, reason, expiry, and invocation/cycle scope.

`--max-tokens N` is a hard policy by default. Before planning any spawn, the
loop excludes implementation runtimes without a declared `token_limit_flag`
and refuses if its implementation pool or selected review role has no capable
runtime; `--dry-run` applies the same gate. If the operator deliberately wants
accounting without provider-side enforcement, pass
`--allow-advisory-tokens`. The loop warns `ADVISORY ONLY`, forwards that choice
to both implementation and review spawns, and still enforces its rolling
window, worker timeout, cycle, idle, and stop-file guards. Operating profiles
persist the same choice as `budgets.allow_advisory_tokens`; omission means hard
mode. Runtime names never imply this capability—the adapter declaration is the
only authority.

### Landing: auto-merge, and the integrator role

Two mechanisms keep "a broken main never happens" true with no human watching:

1. **Controller-owned PR landing, with optional auto-merge.** After a worker commits, the loop pushes the canonical branch and creates or reuses its PR itself, so landing does not depend on the child reaching the last line of its prompt. Persisted `auto_merge=false` leaves that verified PR open for explicit landing. When `auto_merge=true` is deliberately configured, the controller adds `--auto`, which queues GitHub's native auto-merge behind required checks — never a silent local merge over red or pending CI.
2. **The `integrator` role.** A standing, spawnable `rw` reviewer-kind agent (see its row in [ROSTER.md](ROSTER.md)) whose entire charter is release management: sweep open PRs on done tasks, merge the ones with green `gh pr checks`, queue `--auto` on ones still running, and refuse to merge red CI — filing a finding naming the failing check instead. It never implements. Spawn it as a standing backstop wherever a PR might need landing outside the loop's own inline path — a human-triggered spawn wave, a PR opened without `--auto`, a stuck merge — the same merge discipline the loop applies to itself, callable on demand.
