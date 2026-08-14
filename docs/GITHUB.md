# GitHub Issues and Projects

**Status: the bidirectional Issues mirror (the G-series) is implemented** in the `ghmirror` slice (`internal/features/ghmirror/ghmirror.go`) and the PR-enrichment path in the `vcs` slice (`internal/features/vcs/lifecycle.go`). What ships today, and where it still stops:

- **Outbound push (`dacli github push <project>`)** projects local state onto Issues, all behind one disclosure gate and all marker-idempotent (§ 4):
  - **Tasks → issues.** Each task becomes an issue whose body carries the acceptance criteria and a recovery marker; a single `status:<folder>` label (`status:open|active|blocked|done`) mirrors the task's status folder; the issue is assigned to the **project milestone** (§ 9.7); a **done** task's issue is **closed** best-effort; the issue number is written back into the task's `github:` frontmatter block as a backlink.
  - **Decisions → labeled issues.** Each decision note becomes an issue labeled `decision` whose body is the *why* — Chose / Rejected / Because — plus a backlink to the note.
  - **Findings → issue comments.** Each finding note about a task is posted as a comment on that task's issue, idempotent by a per-finding marker so a re-push never duplicates.
- **Inbound pull (`dacli github pull <project>`)** adopts human-authored issues as local tasks: any issue *without* a dacli marker and *not* already mapped to a task seeds a new task (title + body → its Context), with the `github:` block written back so the next pull/push treats it as linked. Idempotent by issue-number mapping, not by editing the remote.
- **`dacli github sync <project>`** = pull then push, so a freshly adopted issue is mirrored back on the same invocation.
- **PR enrichment + verify verdicts (`dacli pr`).** The PR body is assembled from the task's acceptance criteria, its finding notes, and a `Fixes #<issue>` line (so merging closes the mirrored issue). `dacli pr --with-verdicts` additionally leads the body with a loud trust-grade summary + per-finding verdict tally, and posts the same section plus the verify panel's recorded per-seat verdicts as a single PR **review comment**. `dacli pr --draft` opens the PR as a GitHub draft (§ 9.7).
- **Planning artifacts (§ 9.7).** A project maps to a **milestone** (task issues assigned on push); **`dacli github codeowners`** projects role `scope` globs into a `.github/CODEOWNERS`; **`dacli pr --draft`** opens work-in-progress PRs.
- **The disclosure gate (§ 7), in full.** `github link` refuses a public repo without `--allow-public`; consent is recorded *in the project file* (committed, blameable), **per project** even on a shared repo, and visibility is re-checked **live at every push** — a repo flipped public after linking re-trips the gate, and the finding-comment path rides the same gate. Every push is operator-triggered; **nothing publishes automatically** — no `ship`, `commit`, or `spawn` path touches the remote.
- **Not yet**: Projects-v2 fields (§ 2's priority/estimate columns), status changes as `propose-status` *events* (pull adopts issues directly as tasks today rather than recording inbound events per § 3), conflict reconciliation with `remote_updated_at` (§ 5), batching/backoff (§ 6). The `gh` subcommands used are assumptions until `github doctor` grows probes beyond binary+auth+repo+visibility.

`dacli` mirrors its workspace to GitHub Issues through the `gh` CLI, so humans can see and steer agent work from where they already coordinate. Credentials are `gh`'s own — dacli never handles, stores, or prompts for a token — and every `gh` call runs under a 120 s deadline so a wedged request cannot hang the CLI or, under `dacli mcp serve`, the stdio loop.

## Collaboration boundary: shipped CLI, proposed App

### Shipped today: `gh` CLI adapter

The behavior described as shipped in this guide is operator-run CLI behavior. It executes `gh` as the current user, uses that user's existing `gh auth` session, and makes remote calls only when a `dacli github ...`, `dacli pr`, or PR-first integration command is invoked. There is no server, webhook receiver, GitHub App, installation-token exchange, or continuously running GitHub worker in the shipped product.

No App is needed for a developer, a small team, or a self-hosted swarm that already runs in a trusted checkout and can authenticate `gh`. This is the simplest boundary: GitHub sees the operator's identity and permissions; dacli stores no GitHub secret; sync is explicit; and the local markdown record remains usable offline. Prefer this mode unless the workflow actually needs unattended, organization-managed GitHub events.

### Proposed: installation-scoped GitHub App adapter

An App becomes valuable when an organization needs unattended operation across repositories, centrally revocable installations, audit records that distinguish the automation from a human, or event-driven reactions without polling. That adapter is a proposal, not shipped behavior. It would be an additional port onto the same local projection rules, not a replacement source of truth and not permission for GitHub to mutate workspace files directly.

Its authority should be installation-scoped and least privilege: install it only on selected repositories; request issue, pull-request, metadata, and checks permissions only for operations enabled there; keep contents read-only unless a named workflow must push a branch; and do not request organization-wide administration. Installation tokens should be short-lived and exchanged outside agent prompts and transcripts. A deployment that only imports issues needs less authority than one that creates PRs or enables auto-merge, so those capabilities must remain separable.

The event trust boundary matters as much as the credential boundary. A valid webhook signature proves GitHub delivered a payload; it does **not** make the issue body, comment, actor, or linked content trusted instructions. A future receiver must verify the signature and installation/repository identity, reject replays, persist the raw delivery idempotently, and translate allowed actions into attributed dacli events. The workspace owner then applies those events under dacli's existing ownership and grant rules. It must never feed webhook text to an agent as system instructions or let a remote close/comment bypass the proposal boundary described in § 3.

### Docker is an execution envelope

Docker can package the dacli binary, `git`, an agent runtime, and `gh` into a reproducible process boundary. It can also limit mounted credentials, network reach, CPU/memory, and which checkout a worker can write. That makes it a useful **execution envelope**, not a scheduler, trust policy, GitHub identity, or replacement for dacli's grants and path claims. A container still acts with every credential and mounted path it receives.

Docker support is **not shipped**: this repository currently publishes no supported image, Dockerfile, Compose topology, or container lifecycle contract. A containerized or managed runner remains future control-plane work under [GitHub issue #446](https://github.com/mlnomadpy/dacli/issues/446); image construction, credential mounts, UID/file ownership, worktree sharing, and shutdown/recovery still need their own executable contract before dacli can claim support. Likewise, “swarm” below means dacli's coordinated agent workflow; it does not claim Docker Swarm mode support.

## GitHub-first swarm workflow

This is the end-to-end path when humans start and review work in GitHub while dacli keeps the durable execution record:

1. **Map the issue.** Link the project once with `dacli github link <project>` (adding `--allow-public` only after reviewing § 7), then run `dacli github pull <project>`. A human-authored, unmapped issue becomes a local task with its issue number in `github:` frontmatter. Conversely, `dacli github push <project> <task-ref>` creates or adopts the issue for a local-first task.
2. **Claim before writing.** Select ready work with `dacli next`, then claim the task. `dacli spawn --task <ref> --worktree` performs the normal agent path: it assigns an identity and claim and creates the derived `dacli/<seq>-<slug>` branch in an isolated worktree. A manual worker uses `dacli task claim <ref>` and must preserve the same one-owner/path-claim discipline.
3. **Work in the worktree and record evidence.** Make only the claimed changes, add findings/decisions as they become known, run the task's checks, and mark only acceptance criteria actually proved. Commit with `dacli commit "<message>" --task <ref>` so the commit carries agent, role, and task provenance.
4. **Open the PR.** `dacli pr --with-verdicts` pushes the task branch and opens an enriched PR containing acceptance criteria, findings, the verify result, and `Fixes #<mapped-issue>`. `dacli integrate --pr --no-merge` is the wave-oriented equivalent when review must remain manual.
5. **Wait for CI; do not infer.** Leave a pending or red PR open. Use GitHub's checks and `dacli pr status --task <ref>` to distinguish `landing`, `merged`, `orphaned`, and `unknown`; a stale local branch comparison is not proof. `dacli integrate --pr` merges only after reported checks pass, while `--auto` delegates the wait and merge to GitHub auto-merge.
6. **Integrate through dacli.** Once the task is accepted, land it with `dacli merge --task <ref>` or as part of `dacli integrate --pr` / `dacli ship --pr`. This records the landing and cleans up the worktree; a hand-written `git merge` skips that lifecycle.
7. **Close both views.** Merging the PR closes the mapped issue through `Fixes #...`; the local task moves to done through dacli's acceptance/ship lifecycle. If a task was integrated without PR-first closure, the next scoped GitHub push closes the issue best-effort from local done state.
8. **Push both records deliberately.** Run `dacli github push <project> <task-ref>` after landing to project final status plus scoped findings/decisions and backlinks. Preview with `--dry-run` when the disclosure radius is uncertain. Then use `dacli ship --no-accept --no-integrate --push` when the code is already landed and only the configured `dacli-record` branch needs committing and pushing. Git push moves code, `github push` updates the GitHub projection, and `ship` persists the local collaboration trajectory—none happens merely because an agent finished.

---

## 1. The one decision everything else follows from

**Local markdown is the source of truth. GitHub is a projection.**

Not a peer, not a backend, not "the real store with a local cache." A projection, synced explicitly, that can be deleted and regenerated.

This is worth being firm about because the alternative is genuinely tempting — GitHub has identity, permissions, notifications, and a UI, and it would be less code to just put tasks there. Four reasons not to:

1. **`dacli context` is the hot path and must never touch the network.** It runs on every spawn, potentially dozens of times a minute across a tree. Network latency on the core operation would make the tool feel broken, and an offline agent would be a dead agent.
2. **The concurrency model depends on local files.** Contention-freedom comes from every cross-agent write being a new ULID-named file that cannot collide. GitHub is a shared mutable store with no such property, and a tree of agents writing issues concurrently will hit secondary rate limits and interleave badly.
3. **Rate limits are a hard ceiling on fan-out.** The whole design is about running many agents at once. Coupling agent throughput to an API quota caps the thing the tool exists to do.
4. **Git already versions the workspace.** Task history, decision history, and the event log are in commits, reviewable in diffs, bisectable. Issue history is in a database you do not own.

The cost of this choice is that the two can diverge, and § 5 is about handling that honestly rather than pretending sync is atomic.

## 2. Object mapping

| `dacli` | GitHub |
|---|---|
| Project | A GitHub Project (v2), optionally a milestone |
| Task | An Issue |
| Task status folder | Project status field (`open` / `active` / `blocked` / `done`) |
| MoSCoW priority | Project single-select field |
| Three-point estimate | Project number field, carrying `Te` |
| Task dependencies | Issue body task list + `blocked-by` label |
| Risk (rank 1–2) | Issue labeled `risk`, `rank-1` |
| Finding (major) | Issue comment on the task |
| Decision | Issue comment, plus a pinned summary comment |
| Help request | Issue with `needs-answer`, assigned to a human |
| Agent | Not mirrored — agents are ephemeral, humans are not |

Decisions and findings mirror as **comments, not issues**, because they are commentary on work rather than work. Creating an issue per finding turns the tracker into noise within a day. (Findings ship exactly this way — § 9.1; **decisions ship as labeled issues** in the current build, not comments — see the top-of-file status and § 9.1.)

**Agents are deliberately not mirrored.** A tree that spawns forty children would create forty GitHub artifacts representing processes that lived for ninety seconds. The `agent tree` view stays local.

## 3. Sync direction

**Outbound (`dacli` → GitHub)** is the default and covers structure: projects, tasks, status, priority, estimates, risks.

**Inbound (GitHub → `dacli`) arrives as events, not as writes.** This is the piece that makes the whole thing fit:

| Human action on GitHub | Becomes |
|---|---|
| Comment on an issue | `comment` event |
| Closing an issue | `propose-status` event |
| Moving a card in the Project | `propose-status` event |
| Adding a `blocked` label | `block` event |
| Answering a `needs-answer` issue | `answer` event → promoted to a decision note |

A human commenting on GitHub is structurally identical to a child agent appending a finding: an outside party contributing to an object it does not own. The event log already handles exactly that, so inbound sync needs **no new concurrency machinery at all**.

It also preserves the invariant that only an object's owner rewrites it. A human closing an issue does not move a file — it proposes a status change that the owner applies on `dacli sync`. Nothing races.

> **Shipped behavior (G4).** The events model above is the design target; what `dacli github pull` implements today is the first, load-bearing half of it: **a human-authored issue is adopted as a new local task.** `pull` lists every issue (open and closed) via the strongly-consistent list endpoint and, for each issue that carries **no** dacli marker (so it is not our own projection) and is **not** already mapped to a task, calls `store.CreateTask`, seeding the task's Context with a backlink and the issue body and writing the `github:` block back so the issue is never re-imported. Idempotency here is **number-mapping**, not a body marker, precisely because `pull` never edits the remote. Inbound *comments* and remote status changes as `propose-status` events are still the planned continuation — see the top-of-file status.

## 4. Identity and idempotency

Mapping lives in task frontmatter:

```yaml
github:
  issue: 42
  node_id: I_kwDO...
  project_item: PVTI_...
  synced_at: 2026-07-21T18:04:00Z
  remote_updated_at: 2026-07-21T18:02:11Z
```

Local, diffable, and versioned with the task. No separate mapping database to lose. (The current build writes the load-bearing `issue` and `repo` keys into this block; the timestamp/node-id fields above are the fuller design target.)

Every mirrored issue body also carries a marker:

```html
<!-- dacli:t-002 ws:01J8F3K9 -->
```

so that a lost or corrupted mapping is **recoverable by search rather than by duplication**. Duplicate issues are the characteristic failure of naive syncers: a retry after a timeout that already succeeded creates a second issue, and nothing ever notices.

The create path is therefore: check frontmatter → search by marker → only then create. A sync that is interrupted at any point and re-run must converge to the same state. **Recovery reads issue bodies via the strongly-consistent list endpoint and matches the marker by exact substring — deliberately NOT `gh issue list --search`**, whose index is eventually consistent (a fast retry after a create-then-crash would find nothing and duplicate) and tokenized (it strips the angle brackets and colons in the marker). Decisions and finding comments carry distinct marker prefixes (`<!-- dacli-decision:… -->`, `<!-- dacli-finding:… -->`) so the three kinds are never confused for one another or re-adopted across kinds.

## 5. Conflicts

Divergence is normal, so the policy is per-field rather than global:

| Field class | Winner | Why |
|---|---|---|
| Structure (title, body, deps, priority, estimate) | **Local** | Authored by agents against the workspace; GitHub edits to these are overwritten and the sync says so |
| Discussion (comments) | **Remote, inbound only** | `dacli` never edits or deletes a human's comment |
| Status | **Neither — proposal** | Remote status changes become `propose-status` events for the owner to apply (§ 3) |
| Labels | Union | Cheap, and humans use labels for their own purposes |

`remote_updated_at` detects a remote structural edit; the sync reports it as an overwrite rather than silently clobbering someone's typing. (This per-field reconciliation is the design target; the current build's push is authoritative-local for structure and never rewrites a remote comment, and `pull` only *adds* tasks — it never overwrites a remote edit.)

## 6. Rate limits and batching

- **Sync is explicit** (`dacli github sync`) or a background daemon. Never on the path of `context`, `status`, or `spawn`.
- Batched through `gh api --paginate`; Projects v2 needs GraphQL, so field updates go in one mutation per batch rather than one per item.
- Exponential backoff on 403/429, with the remaining quota reported.
- `--dry-run` prints exactly what the run would create, adopt or close and writes nothing — remote or local. It is the correct default habit for a first sync, and is available on every remote-mutating command: `push`, `sync`, `pull`, `project`, `release` and `codeowners` (dacli 294). The preview runs the command's REAL read-and-decide path and only elides the writes, so it can never drift from what the real run would do. Because it writes nothing, `release --dry-run` does not require the `rw` grant a real cut does; the disclosure gate still runs, so a preview that would be refused reports the refusal.
- A large first sync is chunked and resumable — creating 200 issues is a quota event.

`gh` is required and must be authenticated. `dacli github doctor` probes for the binary, auth, repo access, and visibility, and — as with runtimes — **the exact `gh` subcommands used are verified by probing rather than assumed from documentation.** (Batching and backoff are the design target; the current build issues one `gh` call per object under a per-call 120 s deadline. `--dry-run` is implemented across the remote-mutating commands, dacli 294.)

## 7. Safety

**A public repository makes every mirrored artifact public.** Findings and decisions routinely contain internal reasoning, file paths, architecture detail, and occasionally things nobody meant to publish. Mirroring a workspace to a public repo is a disclosure event.

So:

- **`dacli` checks repository visibility and refuses to push to a public repo without explicit per-project confirmation**, recorded in the project file (`github_public_confirmed: true`, written by `github link --allow-public`). Not a flag that can be passed once and forgotten in a script. The check is re-run **live at every push** (`disclosureGate`), so a repo flipped public after linking re-trips the gate, and the finding-comment path rides the same gate.
- Push is **per-project opt-in** and **operator-triggered** — `dacli init` never enables it, and no `ship`/`commit`/`spawn` path publishes anything. `github pull` is inbound and read-only against the remote, so it is deliberately **not** gated: adopting an issue discloses nothing.
- A `private: true` note is never mirrored, in either direction.
- Credentials come from `gh`'s own auth. `dacli` never handles, stores, or prompts for a token.
- **Issue and comment bodies are untrusted input.** Anyone who can comment on a public repo can write text that a subsequent agent reads. Inbound content is data, never instruction — it is attributed to its GitHub author in every brief it reaches, and a `comment` event from an outside account is marked as such. This is the same cross-tree injection problem as [RUNTIMES.md § 12](RUNTIMES.md), with a wider door.

That last point deserves emphasis: enabling inbound sync on a public repo lets strangers put text into your agents' context. That may be acceptable, but it must be a decision rather than a side effect.

## 8. Commands

| Command | Purpose |
|---|---|
| `dacli github doctor` | Probe `gh`, auth, the repo, and its **visibility** (warns if PUBLIC) |
| `dacli github link <project> [--allow-public]` | Bind a project to the current repo; `--allow-public` records the disclosure consent (§ 7) |
| `dacli github push <project>` | Outbound: tasks → issues (+ status label, close-on-done, backlink, **project milestone**), decisions → labeled issues, findings → issue comments |
| `dacli github pull <project>` | Inbound: adopt human-authored issues as local tasks |
| `dacli github sync <project>` | Pull then push |
| `dacli github release <project> <tag>` | Cut a tagged release with **generated notes** on the linked repo (`--notes` overrides, `--target`/`--title`/`--draft`/`--prerelease` pass through). Idempotent: an existing release for the tag is reported, never duplicated. Needs an `rw` grant; **not** disclosure-gated — generated notes are the repo's own merged-PR/commit history, not workspace findings |
| `dacli github codeowners <project> \| --owner <org>` | Emit `.github/CODEOWNERS` from role scopes: each role's `scope` globs → a pattern owned by the role's team handle (`@owner/role`), most-general pattern first so a specific owner overrides a broad one. Local file write, not disclosure-gated (§ 9.7) |
| `dacli pr [--with-verdicts] [--draft]` | Open a PR whose body carries acceptance + findings + `Fixes #issue`; `--with-verdicts` adds a loud trust-grade summary + per-finding verdict tally to the body AND posts it (+ the verify panel's per-seat verdicts) as a PR review; `--draft` opens it as a GitHub draft (CI runs, no review requested until marked ready) |
| `dacli integrate --pr [--auto] [--no-merge] [--merge]` | PR-first: push each done branch, open or reuse an enriched PR (+verdicts). `--auto` sets GitHub auto-merge so GitHub merges after required checks/reviews; red/pending PRs stay open. Effective PR mode fails closed when GitHub is unreachable; it never falls back locally. |
| `dacli ship --pr [--auto] [--no-merge]` | The wave tail in PR-first mode: forwards the flags to `integrate` so a whole wave lands as reviewable PRs; `--auto` is hands-off (GitHub merges each when CI passes) |
| `dacli ship --push --release <tag> --project <p>` | After integrating and pushing a wave, cut a tagged release with generated notes on the project's linked repo (shells `github release`, targeting `--into`). Requires `--push` (so the released ref is on the remote) and refuses `--pr` (whose merges land asynchronously on GitHub's clock, so a release cut now could tag before the wave merges) |
| `dacli pr status <task>` | "Did this land?" — checks gh's PR state (merged/landing/orphaned) before ever falling back to a trunk fetch. **Never** conclude a done task's branch is orphaned from a bare `git merge-base --is-ancestor <branch> main` against your local checkout — see § 9.6. |
| `dacli escalate --github` | File a help request as an issue ([TEAM.md § 3](TEAM.md)) |

`escalate --github` is the piece that was already specified as the terminal escalation hop, and it is the highest-value part of this integration: when no role in the tree owns a problem, it reaches a human where they will actually see it, with a notification, outside the session.

## 9. The G-series in detail

This section maps each shipped command to what it does, verified against `internal/features/ghmirror/ghmirror.go` and `internal/features/vcs/lifecycle.go`.

### 9.1 `github push <project>` — outbound projection

Runs the **disclosure gate first** (`disclosureGate`): the repo's *live* visibility is re-fetched, and a PUBLIC repo with no recorded per-project consent refuses (§ 7). Then, for every task **in the window**:

**The task window (`selectTaskWindow`, dacli 275).** A mature project's done set is hundreds of tasks; mirroring the whole backlog files (or adopts and closes) an issue for every one, so an operator cannot publish a single wave without reaching for raw `gh`. `github push <project> [task-ref...] [--since <dur>]` narrows the mirror to the requested window: the **explicit refs** after the project and/or a **`--since`** cutoff (a duration like `2h`/`90m`, matched against each task's `created` stamp). With **neither**, the full backlog mirrors exactly as before. With **both**, the window is the **union** — the named refs plus everything created since the cutoff. A ref that names **no task** in the project is a **not-found error (exit 4)**, never a silent no-op: an operator who asked to mirror a task must hear it was not found rather than watch push report success having filed nothing. A task with no parseable `created` stamp falls **outside** a `--since` window ("since" means demonstrably created after the cutoff, never "assume recent"). The `--since` cutoff is shared with the `--with-tasks` finding-issue mirror, so both scope identically.

**The window scopes the decision and finding mirrors too (`noteInWindow`, dacli 298).** The window originally narrowed only the *tasks*; the decision mirror ran over **every** decision in the project and the standalone finding-issue mirror was scoped by `--since` alone, so a one-task `github push core 275` still published every other decision — an unbounded disclosure on a public repo, where an unintended publish cannot be taken back. A decision or finding is now **in the window** when it was created at or after the `--since` cutoff (the same temporal rule the task window uses) **or** its `about` field names one of the tasks the **explicit refs** selected. With no window (no refs, zero `--since`) every note is in, so the default whole-project mirror is unchanged. An explicit-ref window therefore publishes only the decisions/findings attached to the named tasks; the rest are reported as `out-of-window` and left unpublished.

**The blast radius is stated first (dacli 298).** Before it creates a single issue, push prints a `plan: will create N task, M decision, K finding issue(s) on <repo>` line — the count of genuinely *new* issues of each kind (a note already mapped or adoptable by marker files nothing), computed from the same in-memory issue-list snapshot the loop uses. On a public repo an operator sees the disclosure size while it can still be aborted.

1. **Resolve the issue.** Read the mapped issue number from the task's `github:` block; if absent, **search by marker** (`searchByMarker`) — a strongly-consistent list-endpoint scan matching `<!-- dacli:<task-id> ws:<ws-id> -->` by exact substring; if the marker still misses, **adopt by exact title** (`markerIndex.findByTitle`, dacli 275) — an issue an operator filed by hand carries the canonical `NNN: <title>` but **no dacli marker**, so it is adopted into the mapping rather than duplicated (the full title must match, never a prefix; the lowest-numbered issue wins a title collision, a deterministic tie-break; the issue body is never edited, so the local mapping written back is what makes the next push skip it, exactly as pull leaves an adopted issue); if still absent, **create** the issue (title `NNN: <title>`, body = marker + "So that…" + Acceptance).
2. **Write the mapping back** — *after* the remote exists, so a crash leaves an adoptable issue, never a dangling mapping.
3. **Status label** (`applyStatusLabel`): give the issue exactly one `status:<folder>` label and strip the other three, so a moved task never accumulates conflicting labels.
4. **Findings → comments** (`mirrorFindings`): each finding note whose `about` names this task is posted as an issue comment led by a per-finding marker `<!-- dacli-finding:<note-id> ws:<ws-id> -->`; a comment already carrying that marker is skipped, so a re-push never duplicates. Comments are fetched once per task, so N findings cost one extra read.
5. **Close on done**: a `done` task's issue is closed best-effort.

After the task loop, **decisions → labeled issues** (`mirrorDecisions`): each **in-window** decision note (see the window note above) becomes an issue labeled `decision`, keyed by `<!-- dacli-decision:<note-id> ws:<ws-id> -->` (a distinct prefix so a decision issue is never adopted as a task mirror), with the same frontmatter → search-by-marker → create idempotency. The body is the **why**: Chose / Rejected / Because + a backlink. Decisions ride the same explicit push and the same already-tripped disclosure gate — never a separate auto-run — and are scoped to the same window as the tasks, so a scoped push never files a decision about a task it did not name.

### 9.2 `github pull <project>` — inbound adoption

Adopts human-authored issues as local tasks (§ 3, *Shipped behavior*). It is **read-only against the remote** — it never edits an issue — so it is **not** gated on public visibility: importing an issue discloses nothing. Idempotent by issue-number mapping.

### 9.3 `github sync <project>`

`cmdPull` then `cmdPush` — each half carries its own linkage and (for push) disclosure checks; running pull first means a freshly adopted issue is mirrored back on the same invocation.

### 9.4 `dacli pr [--with-verdicts]` — PR enrichment and verify verdicts

`prBody` assembles the PR description with no network access (so it is unit-testable): a `Fixes #<issue>` line parsed from the task's own `github:` block (skipped cleanly when unlinked), the task's **Acceptance** section, and a **Findings** section listing every finding note about the task (with severity and trust tags). The opened PR's URL is recorded as a finding so it enters every future brief.

With `--with-verdicts`, `prBody` leads with a loud **`## 🚨 TRUST GRADE: …`** section (`trustGradeSection` in `internal/features/vcs/lifecycle.go`) — first-class and first in the body, ahead of Acceptance/Findings — before `postVerdicts` renders the same section into a PR **review comment** (`gh pr review <branch> --comment`), ahead of the existing per-seat verdict list. The section aggregates this task's finding notes by trust grade (confirmed/unverified/refuted, ordered the same refuted < unverified < confirmed way `internal/brief`'s trust-floor uses — the two share the exported `brief.TrustLabel`/`TrustRank`/`RankTrust` helpers, one vocabulary, no drift), plus a **per-finding panel vote tally** (`N confirmed, M refuted`) joined by claim text to the `verify-verdict:` comment events `dacli verify` records for each panel seat (`VerdictRecord`/`VerdictMarker` in `internal/features/execution/verify.go`); the two slices don't import each other, so the marker string — not an import — is the contract between the verify writer and the PR reader. It reads only data verify and `store.GradeFinding` already collected — no new collection. Posting is **operator-triggered only** (a flag, never automatic), and a post failure is a note, not a hard error — the PR itself already exists and is recorded.

### 9.5 `dacli integrate` — the merge path, and the branch key it lands

**`dacli integrate` is how a done task's branch reaches trunk — not a hand-run `git merge`.** It merges every done branch, in `dacli next` order, one at a time, so a conflict blocks exactly that task (with a finding naming the conflicted files) instead of piling up; `dacli merge --task <ref>` lands a single task's branch and `dacli ship` tails a whole wave. Never `git checkout main && git merge` a task branch by hand: `integrate` is what records the landing, tears down the worktree, and keeps the sequence honest.

**The branch key is `dacli/<seq>-<slug>`** — the task's zero-padded 3-digit sequence number and its slug (`fmt.Sprintf("dacli/%03d-%s", t.Seq, t.Slug)`, `internal/features/vcs/lifecycle.go`). That single derivation is the contract: `dacli spawn --worktree` creates the branch under this name, `integrate`/`merge`/`pr status` find a task's branch by re-deriving it, and the worktree that holds it is keyed on **project + seq + slug** so two same-titled tasks in different projects never collide on one path (dacli 288/215). A branch named anything else is not a task branch `integrate` will find — the key is derived from the task, never typed.

By default `dacli integrate` (and `dacli ship`, which shells it) lands each done task's branch with a **local `git merge`**. `--pr` switches to **PR-first integration**: for every done task with a branch, `prIntegrateTask` (`internal/features/vcs/lifecycle.go`)

1. **pushes** the branch to origin (`dacli push`'s primitive),
2. **opens an enriched PR** — the same `prBody` (acceptance + findings + `Fixes #issue`) as `dacli pr`, and it always posts the **verify-panel verdicts** as a review comment, and
3. **lands** it via `gh pr merge`. Three sub-modes decide *how*:
   - **`--auto`** — `gh pr merge <branch> --auto --merge --delete-branch` sets GitHub's native auto-merge, so GitHub merges the PR the instant its required checks go green and deletes the branch. Nothing merges locally now; the worktree/branch stay put because GitHub owns the pending merge. This is the **hands-off** path: the operator never waits on CI or merges by hand.
   - **default (no flag)** — the **check gate**: `prChecksPass` runs `gh pr checks <branch>` and merges (`--squash`, or `--merge` for a merge commit) **only if every check already passes** (exit 0). A red or pending check leaves the PR **open** and reports it, rather than blindly merging over a failing gate — and so does a repo with **no checks configured at all** ("no checks reported"): an absent gate is reported distinctly from a passed one, not treated as green, so a repo with no CI can't merge everything by default (dacli 216). On a clean merge dacli tears down the local worktree/branch and fast-forwards the local target to the merged remote state.
   - **`--no-merge`** — stops after step 2: the PRs are **left open for human review** and nothing lands.

`--auto` and `--no-merge` both hand the merge to GitHub, so an *offline* failure is **surfaced** rather than silently local-merged behind the operator's back; the default (check-gate) path still falls back to a local merge when GitHub is unreachable so a wave lands offline. Because a merge closes the mirrored issue via the body's `Fixes #<issue>` line, PR-first integration keeps GitHub the source of truth for review while dacli still assembles the body.

**Fail-closed recovery.** In effective `pr` mode, failures during push, PR creation, checks, or merge leave the canonical task branch and existing PR recoverable; they never authorize a local merge that bypasses required checks or reviews. Run `dacli github doctor`, repair `gh auth` or connectivity, inspect `dacli pr status --task <ref>`, and rerun the same command. A deliberate local exception must be explicit (`--landing-mode local`, or loop `--no-pr`) and is recorded as an override.

### 9.6 `dacli pr status <task>` — "did this land?" without false positives

A `--auto` PR merges on **GitHub's clock**, not the reviewer's: it queues auto-merge and returns immediately, then lands whenever CI goes green, possibly minutes later and always asynchronously to whatever the reviewer's checkout is doing. A reviewer or backlog-auditor who checks "did this accepted task's branch actually land?" by running a bare `git merge-base --is-ancestor <branch> main` against **their own local `main`** is comparing against a snapshot from whenever they last fetched — not against GitHub's current state. Two false positives shipped from exactly this mistake: task 157 (task 154's CI change) and task 160 (task 159's fetch/fast-forward fix) were both filed as "accepted+closed but branch orphaned" when in fact the branch's PR was still landing — 159's own branch merged shortly after (PR #268), proving the "orphan" was never real.

**The rule going forward: a branch not (yet) an ancestor of local main is not evidence of anything.** Before calling a done task's branch orphaned, run `dacli pr status <task>` (`checkLanded` in `internal/features/vcs/lifecycle.go`). It answers with one of:

- **`merged`** — GitHub reports the PR MERGED, or (no PR on record at all, e.g. a local `dacli integrate`) the branch is an ancestor of a **freshly fetched** `origin/main`.
- **`landing`** — the PR is **OPEN** on GitHub right now, whether or not `--auto`'s auto-merge is queued yet. This is the state 157 and 160 were wrongly filed under — it is not a defect, and re-checking a few minutes later (or after CI finishes) is the only correct next step, not a re-integration task.
- **`orphaned`** — no PR was ever opened (or it was **closed without merging**) and the branch is still not an ancestor of a fresh `origin/main`. This is the only state that justifies a re-integration task.
- **`unknown`** — gh and a trunk fetch both failed to answer; treat as "can't tell", not as "orphaned".

`checkLanded` always asks `gh pr list --head <branch> --state all` first — GitHub's own PR state is authoritative — and only falls back to a git comparison when no PR is on record, and even then it fetches `origin` first rather than trusting whatever the local checkout already had on disk.

### 9.7 Milestones, draft PRs, and CODEOWNERS — the planning artifacts a real project carries (dacli 224)

Three additions that give a mirrored repo the planning scaffolding a hand-run project maintains by hand:

- **Projects → milestones (`github push`).** A project maps to ONE milestone titled by the project (its human title, falling back to the slug), and every task issue is assigned to it — so the tracker groups a project's work the way GitHub milestones are meant to. `ensureMilestone` (`internal/features/ghmirror/ghmirror.go`) is the load-bearing piece: `gh issue create --milestone <title>` **hard-fails on an unknown milestone**, which would abort the whole push, so it passes `--milestone` ONLY once the milestone is *positively confirmed* present. gh has no `milestone create` verb, so creation is a POST to the REST milestones endpoint followed by a re-list to confirm — a milestone that could not be confirmed (a flaky gh, a create that did not land) is simply skipped, exactly like the best-effort labels, never passed as a poison flag. Assignment to adopted/already-mapped issues is best-effort and idempotent.
- **Draft PRs (`dacli pr --draft`).** Opens the PR as a GitHub draft — CI runs but it is not mergeable and requests no review until marked ready, the work-in-progress PR a real project opens before review. It threads through `openPR` as a single flag; **integration never drafts** (it opens PRs to *land*, `prIntegrateTask` passes `draft=false`), so only the operator-run `dacli pr` sets it.
- **Roles → CODEOWNERS (`github codeowners`).** A real project routes review by mapping code areas to owning teams; dacli already carries that mapping in its roles — each role's `scope` globs are the paths it owns. `github codeowners` projects those scopes into `.github/CODEOWNERS`: each glob → a CODEOWNERS pattern (a trailing `**` is dropped since a directory prefix already matches recursively), owned by the role's team handle `@owner/role` (owner from the linked repo, or `--owner`). Roles sharing a pattern share one line (CODEOWNERS's multi-owner form), and patterns are emitted **most-general-first** so a specific owner overrides a broad one under CODEOWNERS's last-match-wins rule. It writes a **local file** — not an outbound GitHub write — so, like template generation, it is not disclosure-gated; and it **refuses to write a hollow header-only file** when no role declares a scope, rather than reporting success over an empty projection. The mapping (`codeownersEntries`/`codeownersPattern`/`codeownersDoc` in `internal/features/ghmirror/codeowners.go`) is pure and unit-tested without gh or the filesystem.

---

# Obsidian

Unchanged from [DESIGN.md § 9](https://github.com/mlnomadpy/dacli/blob/main/DESIGN.md): **conform to the conventions, ship no plugin.**

The workspace is already a valid vault — YAML frontmatter, `[[wikilinks]]`, folders. Opening the project root works today, and graph view renders the decision and finding link structure for free.

Templates ([TEMPLATES.md](TEMPLATES.md)) generate docs into that layout, so documents created by an agent open as ordinary vault notes with no export step.

Two additions worth making, both zero-integration:

- **Index notes.** Templates generate a `_index.md` per project linking its docs, tasks, and decisions — a map-of-content note, which is how vaults are navigated. Cheap, and it makes the vault browsable rather than just present.
- **Inline field syntax** on tasks (`priority:: must`) alongside the frontmatter, so that anyone running the Dataview community plugin gets live task boards and status queries in Obsidian with no work from us. Optional, additive, and invisible to anyone not using it.

What stays out of scope: a plugin, Canvas file generation, and anything requiring Obsidian to be installed. The vault must remain a side benefit of the format rather than a dependency of it.

## The three surfaces

The clean way to state the whole picture:

- **Obsidian** is where humans read and write documents.
- **GitHub** is where humans coordinate and where work becomes visible outside the session.
- **`dacli`** (CLI and MCP) is where agents work.

One markdown store underneath all three. None of them owns it.
