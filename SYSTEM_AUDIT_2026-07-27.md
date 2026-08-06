# dacli — Full System Audit

**Date:** 2026-07-27 · **Commit:** `c0bbc6d` · **Method:** 8 parallel adversarial audit passes + a clean-room reproduction harness. Every "CONFIRMED (reproduced)" item below was executed live in a throwaway workspace; artifacts cleaned up, the real repo untouched.

---

## Verdict

The codebase is **well-architected and genuinely tested at its core** (atomic writes, event log, foreground process supervision, FSD boundaries — all clean), but it has **three categories of serious problem**:

1. **The read-only grant is not a security boundary.** Four independent, reproduced paths take a `ro` agent to arbitrary command execution as the operator, plus one to arbitrary filesystem write/delete outside the workspace.
2. **Two silent-data-loss / false-signal bugs erode trust in the tool's own output** — a spawn that does no work can be recorded as "done", and any CRLF file (i.e. every file on a Windows clone) loses all frontmatter with no error.
3. **The public docs and landing page make many claims the code does not support** — including a broken primary install path and inflated headline numbers.

Ranked below. P0 = fix before any public release or before running an unattended loop on anything you care about.

---

## P0 — Security: the grant model is advisory only

**Root cause (single):** the `dacli` binary is on the child agent's Bash allowlist (`.dacli/runtimes/cc.md:11`), and several subcommands that mint execution or write paths have **no grant check**. You sandbox the child's shell, then hand it a privileged CLI that rebuilds the sandbox.

| # | Path | Evidence | Status |
|---|---|---|---|
| S1 | `shortcut add --effect read` → `run` = arbitrary shell as operator (effect is self-declared; `cmdAdd` has no grant check) | `shortcuts.go:39`, `shortcut.go:286`, `shortcuts.go:204` | **CONFIRMED (reproduced):** `ro` agent ran `echo … > /tmp/x` as the user |
| S2 | `project add --slug '../../x'` writes a file outside the workspace; `project rm --force` → `os.RemoveAll` on it (explicit `--slug` bypasses `Slugify`; bare `filepath.Join`, no `..` check) | `planning.go:36`, `store.go:91`, `workspace.go:213` | **CONFIRMED (reproduced):** landed a file 4 levels above workspace root |
| S3 | `pr status --into '--upload-pack=<cmd>'` → `git fetch origin <into>` executes `<cmd>` (no grant check on `pr status`; no `--` end-of-options) | `lifecycle.go:298,285`, `gitx.go:234` | **CONFIRMED (reproduced):** `git fetch origin --upload-pack=…` executed |
| S4 | `runtime add` (no grant check) + `env_passthrough` (no denylist) restores `ANTHROPIC_API_KEY` to a child; the "never pass the key" rule is enforced *only by a default preset value*, not a check | `execution.go:79`, `store/runtimefiles.go:112`, `execution.go:65` | CONFIRMED by code |
| S5 | `.dacli/STOP` kill switch is a plain file inside the repo children edit, checked only once per cycle; governor budget in `state.go` is unvalidated `key: value` (`window_spent: 0` defeats the ceiling across restarts) | `governor.go:116`, `orchestration.go:949`, `state.go:100` | CONFIRMED by code |
| S6 | The `dacli` binary itself is at an allowlisted path and is a writable file — an agent with `Write` overwrites it; Claude Code's prefix allowlist then runs it | `.dacli/runtimes/cc.md:11` + `.gitignore:2` | CONFIRMED by code |
| S7 | GitHub write surface without a grant check: `report` (`--repo` fully agent-controlled → exfil to any repo the gh token can write), `escalate --github`, all of `ghmirror` (gated only on one-time public consent, never grant) | `selfreport.go:93`, `collab.go:244`, `ghmirror.go:215+` | CONFIRMED by code |
| S8 | `catalog.disclosureGate(w, repo, p)` **ignores its `repo` param** — checks cwd repo visibility but publishes the roster to the *stored* repo's wiki. Private cwd + public linked repo ⇒ gate passes, roster goes world-readable | `catalog.go:339,271` | CONFIRMED by code (unused param is the tell) |

**Fix shape:** one grant/allowlist gate on the command dispatcher (`cli.go` executor) covering `shortcut add`, `runtime add`, `project add/rm`, `pr status`, `kill`, `report`, `escalate --github`; a `filepath.Clean` + workspace-prefix assertion inside `workspace.dacli()`; `--` end-of-options on every gitx argv; and move the `dacli` binary off the child allowlist (or gate by absolute path to an installed, non-writable location).

**Genuinely clean (do not chase):** no secrets anywhere across 1,394 files / all refs; agent `token_hash` values are one-way SHA-256; no *shell* interpolation in any git/gh call (all `[]string` argv, so classic shell injection and flag-injection-via-title are NOT reachable — the finding-title claim from one pass was checked and rejected); branch/PR names can't inject flags; no `pull_request_target`; no path to change repo visibility/settings.

---

## P0 — Correctness: the tool lies about its own state

### C1. A spawn that does zero work can be recorded as "done" — **CONFIRMED (reproduced)**
`gitx.AddWorktree` creates the branch **at spawn time** (`gitx.go:145`), before the child does anything. So:
1. The post-wait "did it land?" recheck at `orchestration.go:452` tests `branchExists` — now always true; the recovery check is a no-op.
2. Next cycle `prLandStatus` finds no PR and falls to `IsAncestor(branch, origin/main)` (`orchestration.go:625`). A zero-commit branch **is** origin/main's tip → returns `"merged"`.
3. → `accept --force` → task closed, every acceptance box ticked.

Reproduced: `rev-list --count origin/main..dacli/001` = `0`, `merge-base --is-ancestor` = **true**. So a child that OOMs/is killed/refused right after launch ⇒ task recorded done, no commits, no work. This is the exact failure #74/#75 closed, reintroduced via the branch-existence proxy. **Consequence:** the 157 "done" tasks and the derived headline numbers are not a trustworthy completion signal. Fix: gate on `rev-list --count trunk..branch > 0`; treat zero commits as failed.

### C2. Any CRLF file loses ALL frontmatter, silently — **CONFIRMED (reproduced)**
`mdstore.go:317` gates parsing on `HasPrefix(raw, "---\n")`. A CRLF file starts `---\r\n` → branch skipped → `Parse` returns empty `Front`, **no error**. No `.gitattributes`; Git-for-Windows `core.autocrlf=true` ⇒ a Windows clone materializes all 1,109 workspace files as CRLF at once. Reproduced: after CRLF conversion, `task list` looks normal and `doctor` says "no anti-patterns", but `--json` shows `"id": ""` — every object's identity, ownership (`CanMutate`), event correlation, GitHub mapping, priority, and acceptance state silently void. **dacli does not work on Windows despite shipping Windows binaries.** Fix: normalize CRLF on read + add `.gitattributes`.

### C3. Free-text flag values corrupt or inject frontmatter — **CONFIRMED (reproduced)**
`Front.Set` stores verbatim; `Render` emits `key: value` with no escaping (`mdstore.go:477`).
- `--priority $'must\nowner: a-attacker'` → a real second `owner:` key (reproduced).
- `--priority $'must\nplain-text'` → `task add` prints success, then the task is **permanently invisible** — absent from `task list`, `task show` says not-found, `doctor` clean (reproduced).
- ` #` truncates a value (`fix bug #42` → `fix bug`); a line that is exactly `---` splits the file into two frontmatter blocks.
Reachable via `--priority`, `note --origin/--against`, `skill --desc`, `queue --fail`, role/shortcut summaries. `SetList` quotes; `Set` has no counterpart. Fix: quote/reject newlines & control chars in `Set`, mirroring `quoteListElem`.

---

## P1 — Correctness: governor & loop guarantees don't hold

- **The thrash guard can never fire.** `landed` is a trunk-marker delta, but every cycle `recordSelfPR` commits `.dacli` bookkeeping onto trunk (events/tasks are tracked) ⇒ `landed ≥ 1` unconditionally ⇒ `zeroStreak` resets (`governor.go:159`) ⇒ `--no-progress-halt` never trips. It is the *only* stop condition of a perpetual `--yolo` loop. CONFIRMED.
- **`--max-cycles N` does not bound an idle loop.** The `backlog==0` branch never calls `AfterCycle`, which owns the counter ⇒ `loop --max-cycles 1` on an empty backlog runs forever, spawning a paid review agent every 30 min. Tests mask this via `dryRun`. CONFIRMED.
- **`retro` fails every cycle.** The loop runs `retro --project <slug>` but `cmdRetro` requires a positional ref + one of `--well/--bad/--improve` ⇒ exit 2 unconditionally. CONFIRMED.
- **`--into` is never passed to `ship`.** `ship` defaults `--into main` and refuses if `CurrentBranch != into`; the loop resolves the real trunk but never forwards it ⇒ on any `master`/renamed-trunk repo the LAND phase fails silently every cycle. CONFIRMED.
- **`pendingAccept`/`pendingLand` are in-memory; default mode is one cycle/process** ⇒ under non-`--yolo` driving, a task whose PR lands is never `accept --force`d, and the next invocation re-picks it and spawns a *second* implementer onto the existing branch. CONFIRMED.
- **`resolveTrunkBranch` returns the literal `"HEAD"` on detached HEAD**, and prefers a stale local `main` over the real default when `origin/HEAD` is unset (common in CI/shallow clones). CONFIRMED.
- **A transient `git` failure fabricates both a false stall and a false burst** — `trunkMarker` returns 0 on any failure, feeding the thrash guard indistinguishably from real non-progress. CONFIRMED.
- **`dacli wait` checks only the leader PID** (`AliveRecord`), not the group (`GroupAlive` exists, unused) ⇒ the loop proceeds to LAND while children are mid-commit. CONFIRMED.
- **`accept --force` (single ref) verifies nothing** — not a proposal, branch, commit, or `--verify`; root can close any untouched task. CONFIRMED.
- **Concurrent `task add` assigns duplicate seq** (unlocked `max+1` scan) ⇒ both tasks become unaddressable (`FindTask` → "ambiguous"). CONFIRMED.
- **`CloseTask` / sync are non-atomic two-steps**; a crash mid-way double-stamps on retry (`AppendLog` has no idempotence guard, unlike `eventlog.logOnce`). CONFIRMED.
- **`ghmirror.markerIndex.find` sets `loaded=true` before the fetch** ⇒ one transient `gh` failure poisons the index for the whole push ⇒ every issue re-created as a duplicate (the exact crash-recovery case the markers exist for). CONFIRMED.
- **`eventlog` swallows: missing `applied:` key reads as unapplied yet is excluded from every Pending query** ⇒ stuck forever, no signal. `Sync` strands a child's later events after applying its claim (ownership flips mid-pass). CONFIRMED / SUSPECTED respectively.

---

## P1 — Exit-code contract drift (a supervisor branches on these)

- **Exit 5 (Conflict) is documented but unimplemented/unreachable** — no `Conflictf`, no `ExitCode` case.
- **Should be 3 (no-retry), returns 1:** `execution.go:805` ("stalled … do not simply re-run" — the message states the rule the code contradicts), `queues.go:69` (halt), `insight.go:495` ("this command refuses"), `ship.go:227`.
- **Should be 4 (not-found), returns 1:** `lifecycle.go:168/636` (no branch), `skillforge.go:219`, `orchestration.go:178` (no loop state), plus library sites in `prompts`/`team`.
- **Unknown flags silently ignored on ~96% of the CLI.** `Flags.Reject` is called by **4 of 112** handlers ⇒ `task list --statuz open` → exit 0, filter dropped; `status --jsn` → exit 0, wrong output; `next --paralel 2` → exit 0, ignored. For an agent-driven tool this is the worst failure mode: a typo returns success + wrong data. CONFIRMED (reproduced).

---

## P1 — One swallowed error that loses work

`lifecycle.go:654` — on a merge conflict, `store.SaveTask`/`MoveTask` results are `_`-discarded, then the command returns exit 3 "blocked". If either write fails, the task stays in `active/` while the message + event log say blocked ⇒ `next` re-hands it out and a supervisor re-spawns onto the conflicted tree. This is the **only** site (of ~100) that discards a `Save*`/`MoveTask` result.

---

## P2 — Docs & landing page contradict the code (public marketing)

- **Primary install path is dead.** `brew install mlnomadpy/tap/dacli` is the landing-page hero CTA and quickstart step 1, but: tap repo does not exist, `HOMEBREW_TAP_GITHUB_TOKEN` secret is absent (`total_count: 0`), zero tags, zero releases. The `curl .../releases/latest` path 404s too. **Only `go install` works** (verified) — and it embeds the *legacy* dashboard, not the Vue SPA in the screenshot, because `ui/dist` is gitignored. No caveat anywhere warns a new user.
- **Even if you tag, the release job fails at the cask step** — it will build all 6 binaries, publish the GitHub release, then error on the missing tap/token, leaving a release `brew` can't install.
- **Headline numbers are wrong:** "100+ merged PRs" (home.html) / "80+" (README, docs) — actual **77**. "6 bugs it found in its own governor" — unsourced; the string "governor" appears in no finding/DOGFOOD record.
- **Four live doc pages open with "nothing here is implemented"** for shipped subsystems (`MCP.md`, `SPM.md`, `TEAM.md`, `WALKTHROUGH.md`, plus `ARCHITECTURE.md`/`FORMAT.md`/`DESIGN.md` "specification only" headers).
- **Many flags/commands documented that don't exist:** `lint --ambiguity/--strict`, `events compact`, `stage advance --force`, `github sync --dry-run` (the "preview" runs a real sync), `context --depth/--format`, `distill`, `logs -f` (broken — only `--follow`), adapters for codex/gemini/opencode/mock (only claude-code + generic-exec ship), and more.
- **False behavioral claims:** "every command takes `--json`" (3 of 110 do); "dacli does not execute anything" (it runs `run`, `spawn`, `accept --verify`, `gh`); the quickstart's own `dacli context task/001` fails (exit 4 — wrong ref form); "two honest stubs remain" (both implemented — zero `Planned()` call sites).
- **DOGFOOD.md is framed as "real output" but is badly stale** across ~12 numbers (9 vs 118 agents, done 25 vs 157, pending 9 vs 244, etc.).
- **Stale counts** in ROSTER (9 vs 17 roles), MCP tool count (14 vs 16), slice count (10 vs 21), and several FORMAT enums missing shipped kinds (notably `commit`, the most common event).
- **Undocumented shipped commands:** `role bump/show`, `skill bump`, `catalog`, `dashboard`, `github project`, `loop status`, `pr status`. Env var `DACLI_REPORT_REPO` documented nowhere.
- **CI gap:** `docs.yml` path filter omits `overrides/**` ⇒ landing-page-only edits never deploy.

---

## P2 — Dashboard hardening

- **Unauthenticated path traversal:** `?project=../../other-workspace` reads task metadata from any other dacli workspace on the machine (`dashboard.go:127`). Scoped: not arbitrary file read (path must match `<X>/tasks/{status}/*.md`), bind is loopback-only. But `validRunID` one file over is a strict allowlist — `project` just never got the same treatment. CONFIRMED (reproduced by the dashboard pass).
- **No auth, no `Host` validation** ⇒ DNS-rebinding defeats the loopback mitigation; `/api/agents/diff` returns full working-tree diffs, `/api/agents/transcript` full transcripts.
- **`--host` flag silently ignored** (no `Reject`) rather than rejected — fails safe but confusing.

---

## P3 — Hygiene & portability

- **2.1 GB of stale state:** 72 worktrees + 71 merged local branches (vs 5 remote) never reclaimed; the loop never calls `worktree remove` after a task lands. (This is the slowness observed earlier.)
- **80% of the tracked repo is bookkeeping** — 1,109 of 1,394 files are `.dacli/` (224 agents, 329 events, 157 done tasks), unbounded and uncompacted; every clone pulls it. `events compact` is documented but doesn't exist.
- **`site/` is not gitignored** — 4.2 MB of mkdocs output one `git add -A` from being committed.
- **An npm dependency injects Go into the build:** `flatted` ships a `.go` file under `node_modules`, so `go build/test/vet ./...` compile third-party Go; results differ depending on whether `npm ci` ran.
- **Windows portability holes beyond CRLF:** `procmon.go:198` (`ps`), three `sh -c` sites, and a POSIX-only env allowlist (`USERPROFILE` unset ⇒ `claude` auth fails) — so the `--max-rss` reaper and `run`/acceptance-verify all fail on Windows.
- **The lesson matcher matches everything** (`insight.go:312`, single-word overlap) ⇒ all 10 retros attach to every task; `next` emits 10 noise lines per suggestion.
- **MCP `serverInfo.version` hardcoded `"0.3"`** while real version is `buildinfo.Version`.
- **All 15 CI third-party actions float on tags** (no SHA pins); `goreleaser` binary pinned to `latest`; `ci.yml` has no `permissions:` key (safe only by mutable repo default); release runs a weaker gate than CI (no gofmt/vet/frontend) and doesn't trigger on tags.
- **Duplications that have already diverged:** `skillManifest` (case-sensitivity mismatch: catalog vs skillforge), `firstLine` (CRLF/trim), `runGH`/`prLandStatus` vs `checkLanded` (dropped timeout classification + hardcoded `main` fallback), `percentile` ×3 (one copy already lives in the importable `store`).
- **`arch_test.go` gaps:** enforces feature↔feature (clean, 0 violations) but **does not check feature→cli/mcp** at all, has no transitive check (a shared pkg importing a feature is invisible), discards the ReadDir error, and `TestAppLayerStaysThin` uses a hardcoded 4-item deny list. `buildinfo` is imported by a feature but isn't on the sanctioned list.
- **Test coverage is inverted:** 36.2% overall; the 1,844-line child-process spawner `execution` is at **5.8%**; `gates`/`model`/`agentid`/`skills` + 8 slices at 0%; the small pure packages at 85–95%.

---

## What is genuinely solid (verified, don't chase)

- `mdstore.WriteFile` is truly atomic (temp+rename, cleanup on every error path). Unicode/colons/`SetList` round-trip byte-exact.
- `eventlog` has no append-atomicity problem by construction; skips corrupt files without blinding the reader; `apply` is idempotent.
- The **foreground** exec path kills the process *group*, sets `WaitDelay`, drains pipes before `Wait` — careful and correct. `procmon` PID-recycling defense is sound; `KillTree` signals the negative pgid.
- `PushSync` retries exactly once, cannot loop, and correctly declines to mask a protected-branch rejection.
- FSD boundaries: **zero** cross-slice or feature→app imports (including tests).
- All 6 cross-compile targets build clean; `goreleaser check` passes; config correct (6 targets, `CGO_ENABLED=0`, right ldflags). Quality gates genuinely gate (no `continue-on-error`/`|| true`); `eslint --max-warnings 0` non-mutating; frontend build + 73 tests + lint green; `npm audit --omit=dev` = 0; `dist/` sync is moot by design.
- Shortcut *execution* is well-defended (POSIX single-quoting, effect gate, rw floor on ad-hoc) — the hole is `shortcut add` lacking a grant check, not the runner.
- Build, `go vet`, `gofmt`, full test suite: all clean. Clean-room `init→project→task→next→status→doctor` works end-to-end.

---

## Recommended sequence

1. **P0 security** — one dispatcher-level grant gate + workspace-prefix path assertion + gitx `--` separators + move the binary off the child allowlist. Small, localized, closes S1–S8 together.
2. **C1 + C2 + C3** — merge-base commit check; CRLF normalize + `.gitattributes`; `Set` escaping. Each is a few lines and each currently makes the tool misreport its own state.
3. **Governor/loop P1s** — the thrash guard, `--max-cycles` idle bound, `retro` invocation, and `--into` forwarding, before any further unattended runs.
4. **Docs P2** — fix the install CTA (create the tap + token and tag `v0.1.0`, or lead with `go install`), correct the numbers, remove the "not implemented" headers. This is the public face.
5. **P3 hygiene** — worktree/branch GC in the loop, gitignore `site/`, exclude `node_modules` from the Go build, `Reject` on all handlers.
