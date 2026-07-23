# GitHub Issues and Projects

**Status: the bidirectional Issues mirror (the G-series) is implemented** in the `ghmirror` slice (`internal/features/ghmirror/ghmirror.go`) and the PR-enrichment path in the `vcs` slice (`internal/features/vcs/lifecycle.go`). What ships today, and where it still stops:

- **Outbound push (`dacli github push <project>`)** projects local state onto Issues, all behind one disclosure gate and all marker-idempotent (§ 4):
  - **Tasks → issues.** Each task becomes an issue whose body carries the acceptance criteria and a recovery marker; a single `status:<folder>` label (`status:open|active|blocked|done`) mirrors the task's status folder; a **done** task's issue is **closed** best-effort; the issue number is written back into the task's `github:` frontmatter block as a backlink.
  - **Decisions → labeled issues.** Each decision note becomes an issue labeled `decision` whose body is the *why* — Chose / Rejected / Because — plus a backlink to the note.
  - **Findings → issue comments.** Each finding note about a task is posted as a comment on that task's issue, idempotent by a per-finding marker so a re-push never duplicates.
- **Inbound pull (`dacli github pull <project>`)** adopts human-authored issues as local tasks: any issue *without* a dacli marker and *not* already mapped to a task seeds a new task (title + body → its Context), with the `github:` block written back so the next pull/push treats it as linked. Idempotent by issue-number mapping, not by editing the remote.
- **`dacli github sync <project>`** = pull then push, so a freshly adopted issue is mirrored back on the same invocation.
- **PR enrichment + verify verdicts (`dacli pr`).** The PR body is assembled from the task's acceptance criteria, its finding notes, and a `Fixes #<issue>` line (so merging closes the mirrored issue). `dacli pr --with-verdicts` additionally posts the verify panel's recorded verdicts as a single PR **review comment**.
- **The disclosure gate (§ 7), in full.** `github link` refuses a public repo without `--allow-public`; consent is recorded *in the project file* (committed, blameable), **per project** even on a shared repo, and visibility is re-checked **live at every push** — a repo flipped public after linking re-trips the gate, and the finding-comment path rides the same gate. Every push is operator-triggered; **nothing publishes automatically** — no `ship`, `commit`, or `spawn` path touches the remote.
- **Not yet**: Projects-v2 fields (§ 2's priority/estimate columns), status changes as `propose-status` *events* (pull adopts issues directly as tasks today rather than recording inbound events per § 3), conflict reconciliation with `remote_updated_at` (§ 5), batching/backoff (§ 6). The `gh` subcommands used are assumptions until `github doctor` grows probes beyond binary+auth+repo+visibility.

`dacli` mirrors its workspace to GitHub Issues through the `gh` CLI, so humans can see and steer agent work from where they already coordinate. Credentials are `gh`'s own — dacli never handles, stores, or prompts for a token — and every `gh` call runs under a 120 s deadline so a wedged request cannot hang the CLI or, under `dacli mcp serve`, the stdio loop.

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
- `--dry-run` prints the planned mutations and is the correct default habit for a first sync.
- A large first sync is chunked and resumable — creating 200 issues is a quota event.

`gh` is required and must be authenticated. `dacli github doctor` probes for the binary, auth, repo access, and visibility, and — as with runtimes — **the exact `gh` subcommands used are verified by probing rather than assumed from documentation.** (Batching, backoff, and `--dry-run` are the design target; the current build issues one `gh` call per object under a per-call 120 s deadline.)

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
| `dacli github push <project>` | Outbound: tasks → issues (+ status label, close-on-done, backlink), decisions → labeled issues, findings → issue comments |
| `dacli github pull <project>` | Inbound: adopt human-authored issues as local tasks |
| `dacli github sync <project>` | Pull then push |
| `dacli pr [--with-verdicts]` | Open a PR whose body carries acceptance + findings + `Fixes #issue`; `--with-verdicts` posts the verify panel's verdicts as a PR review |
| `dacli integrate --pr [--auto] [--no-merge] [--merge]` | PR-first: push each done branch, open an enriched PR (+verdicts). `--auto` sets GitHub auto-merge (`gh pr merge --auto --merge --delete-branch`) so GitHub merges on CI green; the default merges only PRs whose `gh pr checks` already pass, leaving red/pending ones open; `--no-merge` stops for review; falls back to a local merge if GitHub is unreachable |
| `dacli ship --pr [--auto] [--no-merge]` | The wave tail in PR-first mode: forwards the flags to `integrate` so a whole wave lands as reviewable PRs; `--auto` is hands-off (GitHub merges each when CI passes) |
| `dacli escalate --github` | File a help request as an issue ([TEAM.md § 3](TEAM.md)) |

`escalate --github` is the piece that was already specified as the terminal escalation hop, and it is the highest-value part of this integration: when no role in the tree owns a problem, it reaches a human where they will actually see it, with a notification, outside the session.

## 9. The G-series in detail

This section maps each shipped command to what it does, verified against `internal/features/ghmirror/ghmirror.go` and `internal/features/vcs/lifecycle.go`.

### 9.1 `github push <project>` — outbound projection

Runs the **disclosure gate first** (`disclosureGate`): the repo's *live* visibility is re-fetched, and a PUBLIC repo with no recorded per-project consent refuses (§ 7). Then, for every task in the project:

1. **Resolve the issue.** Read the mapped issue number from the task's `github:` block; if absent, **search by marker** (`searchByMarker`) — a strongly-consistent list-endpoint scan matching `<!-- dacli:<task-id> ws:<ws-id> -->` by exact substring; if still absent, **create** the issue (title `NNN: <title>`, body = marker + "So that…" + Acceptance).
2. **Write the mapping back** — *after* the remote exists, so a crash leaves an adoptable issue, never a dangling mapping.
3. **Status label** (`applyStatusLabel`): give the issue exactly one `status:<folder>` label and strip the other three, so a moved task never accumulates conflicting labels.
4. **Findings → comments** (`mirrorFindings`): each finding note whose `about` names this task is posted as an issue comment led by a per-finding marker `<!-- dacli-finding:<note-id> ws:<ws-id> -->`; a comment already carrying that marker is skipped, so a re-push never duplicates. Comments are fetched once per task, so N findings cost one extra read.
5. **Close on done**: a `done` task's issue is closed best-effort.

After the task loop, **decisions → labeled issues** (`mirrorDecisions`): each decision note becomes an issue labeled `decision`, keyed by `<!-- dacli-decision:<note-id> ws:<ws-id> -->` (a distinct prefix so a decision issue is never adopted as a task mirror), with the same frontmatter → search-by-marker → create idempotency. The body is the **why**: Chose / Rejected / Because + a backlink. Decisions ride the same explicit push and the same already-tripped disclosure gate — never a separate auto-run.

### 9.2 `github pull <project>` — inbound adoption

Adopts human-authored issues as local tasks (§ 3, *Shipped behavior*). It is **read-only against the remote** — it never edits an issue — so it is **not** gated on public visibility: importing an issue discloses nothing. Idempotent by issue-number mapping.

### 9.3 `github sync <project>`

`cmdPull` then `cmdPush` — each half carries its own linkage and (for push) disclosure checks; running pull first means a freshly adopted issue is mirrored back on the same invocation.

### 9.4 `dacli pr [--with-verdicts]` — PR enrichment and verify verdicts

`prBody` assembles the PR description with no network access (so it is unit-testable): a `Fixes #<issue>` line parsed from the task's own `github:` block (skipped cleanly when unlinked), the task's **Acceptance** section, and a **Findings** section listing every finding note about the task (with severity and trust tags). The opened PR's URL is recorded as a finding so it enters every future brief.

With `--with-verdicts`, `postVerdicts` renders the task's recorded **verify-panel verdicts** into a PR review comment (`gh pr review <branch> --comment`). The verdicts are read from the `verify-verdict:` comment events that `dacli verify` records for each panel seat (`VerdictRecord`/`VerdictMarker` in `internal/features/execution/verify.go`); the two slices don't import each other, so the marker string — not an import — is the contract between the verify writer and the PR reader. Posting is **operator-triggered only** (a flag, never automatic), and a post failure is a note, not a hard error — the PR itself already exists and is recorded.

### 9.5 `dacli integrate --pr` / `dacli ship --pr` — PR-first integration

By default `dacli integrate` (and `dacli ship`, which shells it) lands each done task's branch with a **local `git merge`**. `--pr` switches to **PR-first integration**: for every done task with a branch, `prIntegrateTask` (`internal/features/vcs/lifecycle.go`)

1. **pushes** the branch to origin (`dacli push`'s primitive),
2. **opens an enriched PR** — the same `prBody` (acceptance + findings + `Fixes #issue`) as `dacli pr`, and it always posts the **verify-panel verdicts** as a review comment, and
3. **lands** it via `gh pr merge`. Three sub-modes decide *how*:
   - **`--auto`** — `gh pr merge <branch> --auto --merge --delete-branch` sets GitHub's native auto-merge, so GitHub merges the PR the instant its required checks go green and deletes the branch. Nothing merges locally now; the worktree/branch stay put because GitHub owns the pending merge. This is the **hands-off** path: the operator never waits on CI or merges by hand.
   - **default (no flag)** — the **check gate**: `prChecksPass` runs `gh pr checks <branch>` and merges (`--squash`, or `--merge` for a merge commit) **only if every check already passes** (exit 0; "no checks reported" counts as passing). A red or pending check leaves the PR **open** and reports it, rather than blindly merging over a failing gate. On a clean merge dacli tears down the local worktree/branch and fast-forwards the local target to the merged remote state.
   - **`--no-merge`** — stops after step 2: the PRs are **left open for human review** and nothing lands.

`--auto` and `--no-merge` both hand the merge to GitHub, so an *offline* failure is **surfaced** rather than silently local-merged behind the operator's back; the default (check-gate) path still falls back to a local merge when GitHub is unreachable so a wave lands offline. Because a merge closes the mirrored issue via the body's `Fixes #<issue>` line, PR-first integration keeps GitHub the source of truth for review while dacli still assembles the body.

**Offline fallback (documented, never silent).** If GitHub is **unreachable** — a network failure at push or at PR-open, detected by `isNetworkErr` scanning gh/git output — integrate **warns and falls back to the local `git merge`** path so a wave still lands offline. The one exception is `--no-merge`: the operator explicitly asked for a PR, so an offline failure is **surfaced as an error** rather than silently local-merged behind their back. A **non-network** failure (a protected branch, bad auth, a dirty tree) is always surfaced — never mistaken for an offline condition. Like every outward vcs command, `--pr` is **gated behind an `rw` grant** and is **operator-triggered** (a flag, never automatic).

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
