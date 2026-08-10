# dacli — Full System Audit

**Date:** 2026-08-06 · **Commit:** `66eba4b` (main) · **Method:** six independent read-only audits (core data, process lifecycle, performance, CLI/Go practices, security/grant model, architecture) plus direct measurement against this workspace. Every finding below was verified against current source; the July 27 audit's items were re-checked before being re-reported, and fixed ones are excluded.

---

## Verdict

The engineering *culture* here is genuinely strong — layering rules enforced by tests rather than comments, doc-comments that cite the incident each guard exists for, an exit-code contract, benchmarked de-quadratification, careful crash-ordering in the event log. `go vet` and `gofmt` are clean, the suite passes under `-race`, and the test-to-production ratio is near 1:1 (29.8k / 31.5k LOC).

Three structural problems undercut it:

1. **Enforcement is opt-in at ~90 call sites instead of at the dispatcher.** Every P0 in this report is a guard that exists and works, sitting next to a sibling command that never got it. The July audit predicted this exact failure mode; it recurred in the commands added since.
2. **Coordination truth lives on git branches.** Task status, claims, and the agent registry diverge per branch, so closes vanish until record PRs merge and the loop re-picks finished work. Every recurring incident — issue #382, orphaned closes, worktree shadow workspaces — is this one category error in different clothes.
3. **The loop's most important state is in process memory, in a mode that exits every cycle.** Accept-on-merge-only, hold-push-while-in-flight, and token caps all hold in `--yolo` and quietly evaporate in the documented default.

**Ranked below. P0 = fix before the next unattended run.**

---

## Resolution — 2026-08-09

**Every finding in this report is fixed and merged**, across PRs #389–#405. This
section was added after the fact so the document does not go on describing
problems that no longer exist; the findings below are left in their original
wording, because the reasoning is the part worth keeping.

Verified against `main` at `25317cc`, by running the escapes as a real
read-only agent rather than by reading the diffs:

| Finding | Now |
|---|---|
| S1 blanked `DACLI_AGENT` → root/rw | refused: *"set but empty (lost agent identity)"* |
| S2 `--cooperative` ro→rw child | `spawn needs an rw grant` |
| S3 `shortcut promote` ungated | `shortcut promote needs an rw grant` |
| S4 `catalog --out <abs>` | refused by grant AND, as root, by the path guard |
| P1 six `github` verbs | `github push needs an rw grant` |
| P1 `agents --reap` | `reaping an agent (--reap) needs an rw grant` |

The three structural problems above are addressed at their root rather than
per-instance:

1. **Enforcement moved to the dispatcher** — three times over: the rw grant
   (`Command.Mutates`), `--json`, and unknown-flag rejection. Each has an
   invariant test that fails on any unclassified command, because the same
   guard had already drifted twice by convention.
2. **Coordination truth is off the branch** — `.dacli` is gitignored by
   default from `new`, `init` and `adopt`, coupled to a record branch so the
   history is preserved rather than deleted.
3. **The loop's state is durable** — the landing ledger and token ceiling
   persist across checkpoints, so the guarantees hold in the documented
   default and not only under `--yolo`.

### Corrections to this report

Two claims in the loop-audit follow-up were **wrong**, found by re-reading the
code before implementing them, and are recorded here rather than quietly
dropped — a wrong finding left standing eventually gets implemented:

- *"`team assign` is bypassed; implRole is fixed per wave."* Capacity routing
  was already wired (`orchestration.go:645-677`, dacli 233). The claim
  described pre-233 behavior.
- *"Stage gates are never consulted."* `advanceStages()` runs every cycle.

What both were groping at was real and is fixed: routing and critical-path
order both degrade silently when a task has no estimate, and nothing sized one.

### Not fixed, deliberately

- **`gitTaskSeqCeiling` memoization** (~200ms under the seq lock). Implemented,
  then reverted: a commit can land between two `CreateTask` calls in one
  process, and a stale ceiling means two tasks share a seq — silent corruption
  rather than a slow command. The cross-branch seq tests caught it.
- **Two tasks with no `completed by` stamp** (084, 279). Historical data from
  before the close-path fixes; stamping them retroactively would fabricate a
  record. Calibration has two fewer samples out of ~90.

---

## P0 — Security: guards that exist, on commands that never call them

| # | Finding | Evidence | Attack path |
|---|---|---|---|
| **S1** | **A blanked `DACLI_AGENT` still resolves to `a-root` with `rw`.** Task 288's fix is written, tested — and was **not on main** at audit time (verified: `git merge-base --is-ancestor` failed). | `internal/agentid/agentid.go:78-82` — `os.Getenv` collapses *unset* and *set-but-empty* | Any wrapper or shell that blanks the var turns a `ro` child into root, unlocking every `RequireRW` and the `accept --force` any-task override |
| **S2** | **`--cooperative` lets a `ro` caller spawn a write-capable child.** `spawn`/`verify` check WIP, seniority, phase, budget, taint, claims — never the *caller's grant*. | `internal/features/execution/execution.go:1213-1217`, `:551`; `verify.go:33` | `dacli spawn --runtime cc-rw --grant ro --cooperative` → child gets `Edit,Write,Bash(go:*)` → chains into overwriting the allowlisted binary |
| **S3** | **`shortcut promote` is the ungated twin of `shortcut add`.** The promoter re-declares the effect. | `internal/features/shortcuts/shortcuts.go:75-127` vs the gate at `:55` | Re-label a destructive command `--effect read`; `shortcut.Guard` then permits it for any grant, and `dacli run` executes it via `sh -c` as the operator |
| **S4** | **`catalog --out <abs>` is an ungated arbitrary-path write.** `SafeRelPath` exists and is used elsewhere; not here. | `internal/features/catalog/catalog.go:43-50`, `:97-100` | `dacli catalog --out ~/.claude/CLAUDE.md` from a `ro` agent overwrites any file the operator's uid can write |

**Common fix:** put the capability check on the dispatcher with a per-command declaration on `clikit.Command` — the same mechanism that fixed `--json` centrally. Per-handler discipline has now failed twice.

---

## P1 — Consent, correctness, and the unattended loop

**Six of seven `github` subcommands write to GitHub with no grant check** (`ghmirror.go:152,200,892`, `project.go:391`, `codeowners.go:133`). `github release` is the only one that got it right — copy its shape. The `disclosureGate` is a *disclosure* control, not consent: it fires only on `Visibility == "PUBLIC"`, so **nothing gates a write to a private repo** — precisely the shape of the incident in issue #382 item 6.

**`dacli agents --reap` kills whole process trees with no `rw` gate**, while `kill` enforces one (`execution.go:1711,1788` vs `:2061`). A `ro` child can SIGKILL every sibling agent.

**`worktree remove` / `prune` destroy branches and uncommitted work ungated** (`vcs/lifecycle.go:142,167`) while every sibling in the same file is gated.

**The env-passthrough denylist guards the writer, not the reader** (`execution.go:1253-1258` vs `:128`). Any runtime file reaching disk another way has `env_passthrough` honored verbatim. The list also omits `AWS_*`, `GITHUB_TOKEN`, `GH_TOKEN`, `OPENAI_API_KEY`.

**Lost updates can permanently erase a durable record.** Every mutation is a whole-file rewrite with no object lock; the seq lock covers only seq allocation. Two processes of the *same identity* (the loop's auto-sync and the operator's `dacli sync`) interleave, and a Log line — a claim, a finding, a `completed by` stamp — is lost while its event is already marked applied. Note this is a **cross-process** race, so `-race` and CI cannot see it by construction.

**A task file that fails to parse vanishes from every list, including `doctor`** (`store.go:823-827` — bare `continue`). Its seq is invisible to the allocator, so the next `task add` can reissue it.

**The loop's landing ledger dies every cycle.** `pendingAccept`/`pendingLand` are in-memory (`orchestration.go:305`) and the default non-`--yolo` mode returns after each cycle. On restart: merged PRs are never accepted, in-flight tasks are rebuilt, and the record push advances main under queued PRs. The `--window-tokens` **ceiling** is likewise unpersisted, so a restart without the flag runs uncapped, silently.

**Stale-branch wedge (three cooperating defects).** `PushSync` can never rebase a worktree child's branch (`gitx.go:330-338` checks the *main* checkout's HEAD); `stillPending` equates "remote ref exists" with "PR in flight" (`orchestration.go:707-713`); and the `orphaned` retry path only logs, reusing the branch at its old tip. Result: a leftover `origin/dacli/NNN` holds the record push forever and the loop churns. *Observed live in this session.*

**Unknown-flag rejection has regressed to ~25 handlers**, including mutating ones. Live: `dacli task block 001 --whyy "reason"` exits 0 with an empty reason. Also `--help` is not a flag anywhere — on ungated handlers it is silently dropped and **the command executes** (`task claim 001 --help` claims the task).

**The loop's budget knobs re-create the defect the kernel was built to kill.** `atoiDefault` (`orchestration.go:1493`) is the `Sscanf`-with-ignored-error pattern `Flags.Int`'s own doc-comment condemns. `loop --window-tokens garbage` → 0 → the guard requiring `> 0` never fires → **an operator who asked for a cap runs uncapped**. `--window-tokens 50k` parses as 50.

**A bool flag followed by a positional silently swallows it** (`clikit.go:162-179`): `github push p --dry-run 001 002` turns dry-run **off** and mutates the remote.

---

## P2 — Performance

Local read paths are in good shape (≈0.1 s for `status`/`next`/`doctor` at 303 tasks). The waste is concentrated in remote fan-out and monotonic corpora.

- **`github push` fires ~7 gh subprocesses per task, unconditionally** (`ghmirror.go:454-467`, `:1357-1366` — five separate calls just for status labels). At ~300 mirrored tasks that is **~2,100 gh invocations for a no-op re-push**: 10–17 minutes of pure churn. Fix: collapse the label edits into one call and diff against the snapshot the push already fetches.
- **`github project` issues 2 unconditional GraphQL mutations per item per sync** (~580 per run).
- **`Sync` rescans every finding note per finding event** (`sync.go:131` → `store.go:1463`) — O(events × notes).
- **`brief.Assemble` re-walks all done tasks and all 307 run dirs** via `CalibrationSamples` — 9.9 ms of its 17.4 ms, paid per spawn, ×4 per wave, when the caller already holds the task list.
- **`CreateTask` runs `git log --all --name-only` (~200 ms) per creation, under the seq lock** — a 20-task decompose spends 4 s in git alone.
- **Every dacli invocation shells out to `git rev-parse --git-common-dir`** (~10 ms of `status`'s 27 ms) where an `os.Stat` would do.

---

## P3 — Architecture: the three structural calls

### 1. Coordination state must come off the branch

The system has **already routed around this twice**: `workspace.Find` redirects any worktree to the main checkout's `.dacli` because "a worktree checks out a git-tracked snapshot that is stale the moment the branch was cut," and loop/governor state lives in unversioned `.dacli/loop/`. The design is refuting itself in its own comments. Measured cost today: each of 9 worktrees carries a **6.5 MB stale shadow** of the workspace (98 MB total), and 1,689 tracked files churn through every record PR.

**Decision taken (2026-08-06):** `.dacli/` becomes gitignored by default, **coupled** with a record branch that is on by default — the mechanism already exists (`gitx.CommitPathToBranch`, `dacli new --gitignore-workspace`, `ship --record-branch`) and is already tested against a gitignored workspace. It is opt-in today for one documented reason: gitignore *without* a record branch silently loses history. Coupling them removes that objection. This deletes the record-PR class entirely.

### 2. The event log is a ratchet, by construction

**203 of 234 pending events are `commit` events — 100% of every commit event ever written.** `Append` stamps every event `applied: false` at birth (`eventlog.go:82`), and `apply()`'s switch has cases for claim, release, finding, propose-status, comment and block — **none for commit or run** (`sync.go:236`). They are born pending and can never leave. So `dacli status` nags "run `dacli sync` to materialize" for a condition `sync` can never clear; running it applies zero. *(Verified in practice at the start of this session.)*

`pending` is an overloaded tri-state: awaiting materialization / informational / orphaned — and two of the three have no exit. Fix: split journal events (born terminal, no `applied` field) from mailbox events (lifecycle + named consumer), and add `sync --report` for orphans.

### 3. The loop needs a cycle journal, not a workflow engine

It is already ~70% a durable state machine — every phase is a re-exec whose effects land in the store, and the governor already checkpoints with refuse-on-corrupt validation. The gap is **one struct wide**: persist `pendingAccept`/`pendingLand` and the `WindowTokens` ceiling into the existing snapshot, and retire agents at run exit so WIP derives from `procmon` liveness instead of 312 never-retired agent files.

**Layering note:** the no-cross-import rule genuinely holds (verified with `go list`, enforced by `arch_test.go`) — rare and worth credit. But it is paid for in the fragilest currency available: the loop composes slices by **re-exec'ing dacli and scraping human stdout**, the exact practice `DESIGN.md` declares lost. `--json` exists; the loop is the one client not using it.

**Scale ceiling:** ~1–2k tasks, ~10k events, ~50 concurrent agents. The event scan breaks first — it is per-*action*, not per-task, and `DESIGN.md`'s "performance never binds" is already false for it.

---

## Recommended order

1. **Land task 288** (P0 identity escalation — the fix exists, unmerged). *Done during this audit: conflict resolved, suite green, re-queued.*
2. **Dispatcher-level capability checks** — closes S2/S3/S4, the ghmirror family, `agents --reap`, `worktree remove`, `role add` in one change instead of twelve.
3. **Cycle journal + agent retirement** (S effort) — makes the loop crash-safe and stops WIP leaking.
4. **Journal/mailbox event split** (S effort) — turns the pending count into a health signal instead of a ratchet.
5. **`.dacli` ignored + record branch on by default** — deletes the record-PR and branch-divergence class.
6. **Central flag declaration** driving `Reject`, `--help`, and usage from one source; replace `atoiDefault`/`parseDurDefault` with the refusing clikit readers.
7. **Diff-before-write in ghmirror** — turns a no-op push from ~17 minutes into seconds.

---

## CI gaps worth closing

- **No `-race`** in CI. (Would not have caught the lost-update race — that one is cross-process — but it is the cheapest guard available.)
- **No coverage measurement**, despite dacli's own `tdd` template gating on a coverage floor. The tool does not apply its own quality gate to itself.
- **`reject_flags_test.go` spot-checks 3 handlers**, which is how 25 drifted. The fix is a table over the whole command surface, in the style `exitcode_invariant_test.go` already uses.
