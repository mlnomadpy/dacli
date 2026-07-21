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
