# GitHub Issues and Projects

**Status: specification. Nothing here is implemented.**

`dacli` mirrors its workspace to GitHub Issues and Projects through the `gh` CLI, so humans can see and steer agent work from where they already coordinate.

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

Decisions and findings mirror as **comments, not issues**, because they are commentary on work rather than work. Creating an issue per finding turns the tracker into noise within a day.

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

Local, diffable, and versioned with the task. No separate mapping database to lose.

Every mirrored issue body also carries a marker:

```html
<!-- dacli:t-002 ws:01J8F3K9 -->
```

so that a lost or corrupted mapping is **recoverable by search rather than by duplication**. Duplicate issues are the characteristic failure of naive syncers: a retry after a timeout that already succeeded creates a second issue, and nothing ever notices.

The create path is therefore: check frontmatter → search by marker → only then create. A sync that is interrupted at any point and re-run must converge to the same state.

## 5. Conflicts

Divergence is normal, so the policy is per-field rather than global:

| Field class | Winner | Why |
|---|---|---|
| Structure (title, body, deps, priority, estimate) | **Local** | Authored by agents against the workspace; GitHub edits to these are overwritten and the sync says so |
| Discussion (comments) | **Remote, inbound only** | `dacli` never edits or deletes a human's comment |
| Status | **Neither — proposal** | Remote status changes become `propose-status` events for the owner to apply (§ 3) |
| Labels | Union | Cheap, and humans use labels for their own purposes |

`remote_updated_at` detects a remote structural edit; the sync reports it as an overwrite rather than silently clobbering someone's typing.

## 6. Rate limits and batching

- **Sync is explicit** (`dacli github sync`) or a background daemon. Never on the path of `context`, `status`, or `spawn`.
- Batched through `gh api --paginate`; Projects v2 needs GraphQL, so field updates go in one mutation per batch rather than one per item.
- Exponential backoff on 403/429, with the remaining quota reported.
- `--dry-run` prints the planned mutations and is the correct default habit for a first sync.
- A large first sync is chunked and resumable — creating 200 issues is a quota event.

`gh` is required and must be authenticated. `dacli github doctor` probes for the binary, auth, repo access, and Projects v2 scope, and — as with runtimes — **the exact `gh` subcommands used are verified by probing rather than assumed from documentation.**

## 7. Safety

**A public repository makes every mirrored artifact public.** Findings and decisions routinely contain internal reasoning, file paths, architecture detail, and occasionally things nobody meant to publish. Mirroring a workspace to a public repo is a disclosure event.

So:

- **`dacli` checks repository visibility and refuses to sync to a public repo without explicit per-project confirmation**, recorded in the project file. Not a flag that can be passed once and forgotten in a script.
- Sync is **per-project opt-in**. `dacli init` never enables it.
- A `private: true` note is never mirrored, in either direction.
- Credentials come from `gh`'s own auth. `dacli` never handles, stores, or prompts for a token.
- **Issue and comment bodies are untrusted input.** Anyone who can comment on a public repo can write text that a subsequent agent reads. Inbound content is data, never instruction — it is attributed to its GitHub author in every brief it reaches, and a `comment` event from an outside account is marked as such. This is the same cross-tree injection problem as [RUNTIMES.md § 12](RUNTIMES.md), with a wider door.

That last point deserves emphasis: enabling inbound sync on a public repo lets strangers put text into your agents' context. That may be acceptable, but it must be a decision rather than a side effect.

## 8. Commands

| Command | Purpose |
|---|---|
| `dacli github doctor` | Probe `gh`, auth, repo access, scopes |
| `dacli github link` | Bind a project to a repo and Project (v2) |
| `dacli github sync [--dry-run]` | Bidirectional sync per § 3 |
| `dacli github pull` | Inbound only: fetch remote changes as events |
| `dacli github push` | Outbound only |
| `dacli escalate --github` | File a help request as an issue ([TEAM.md § 3](TEAM.md)) |

`escalate --github` is the piece that was already specified as the terminal escalation hop, and it is the highest-value part of this integration: when no role in the tree owns a problem, it reaches a human where they will actually see it, with a notification, outside the session.

---

# Obsidian

Unchanged from [DESIGN.md § 9](../DESIGN.md): **conform to the conventions, ship no plugin.**

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
