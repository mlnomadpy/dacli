# dacli builds dacli

dacli is developed using dacli. Its own remaining work lives in `.dacli/`
(committed to this repo), picked in `dacli next` order, each task claimed,
verified against its acceptance criteria, and retro'd through the tool. One
feature was hardened by a real spawned opus reviewer; several bugs were caught
by dogfooding that the test suite had blessed.

Most of `main`'s history lands through a task branch (`dacli/<seq>-<slug>`) and
`dacli integrate` / `dacli merge` / `dacli ship`, never a hand-run `git merge` —
see [*Which commits are agent-authored, and how to tell*](#which-commits-are-agent-authored-and-how-to-tell)
below for the exact, regenerable count and the mechanism. Not every merged PR
is agent work, though: some are a maintainer working through an interactive
session rather than a spawned dacli agent (same section). The commits before
task attribution existed were authored directly, during the initial build-out
(see *History note*).

## The reproducible fixture

One command takes an empty repository to shipped code, unattended:

```bash
./scripts/selfhost-fixture.sh
```

It plans a task, spawns an agent into its own worktree, lets the agent commit
and report through `task done`, and then lets **the tool** close its own loop —
`sync` and `ship`, with nothing reconciled by hand. It ends by asserting the
OUTCOME rather than the calls: the code is on trunk, the shipped code passes
its own tests, the task closed with its boxes checked, and trunk actually
advanced.

That last check is the one the whole fixture exists for. Every step above can
report success while doing nothing, which is this project's most expensive
failure class — a trunk that never moved catches all of them at once.

The "agent" is a shell script, so the run is offline and deterministic. What is
being proven is dacli's coordination; a real model would only add variance to a
question that is not about the model.

`TestE2EFixtureRepoGoesFromEmptyToShipped` runs the same arc in CI on every
push. Both exist deliberately: the test proves the arc continuously, the script
lets you watch it and leaves a workspace on disk to poke at.

**It has caught real refusals on the way to working**, each of them correct:
`dacli commit` refusing to commit on trunk (agents need `--worktree`), the
claim guard refusing files the agent had not declared, and a spawned agent
being refused permission to check its own acceptance boxes. Those are the
guards this repo added one incident at a time, and the fixture walks straight
through all of them.


## Commits are authored by dacli agents

Development commits are made through `dacli commit`, so `git log` and
`git blame` answer *which agent, in what role* wrote each line — the same
attribution any team using dacli gets:

```
$ git log --format='%an  <%ae>'
a-khwzk4bfr6 (maintainer)  <a-khwzk4bfr6@agent.dacli>
```

The flow, which is exactly what the `git_workflow` prompt tells every rw
agent — the branch is keyed on the task (`dacli/<seq>-<slug>`) and the landing
goes through `dacli`, never a hand-run `git merge`:

```
git checkout -b dacli/<seq>-<slug>
DACLI_AGENT=<maintainer-token> dacli commit "<what changed>" --task <ref>
dacli integrate --tasks <ref> --into main   # dacli lands the branch; attribution preserved
```

`dacli integrate` (or `dacli merge --task <ref>` for a single task, `dacli ship`
for a whole wave) is the merge path — it merges in `dacli next` order, blocks a
conflicting task with a finding instead of leaving a half-merged tree, and keeps
the record of what landed ([GITHUB.md § 9.5](GITHUB.md)). A hand `git checkout
main && git merge` bypasses all of that and is not how a task branch reaches
trunk.

`dacli commit` refuses to commit on the default branch (the git-discipline
rule, enforced not just prompted), sets the git author to the agent and role,
and stamps three trailers (`internal/features/vcs/vcs.go:150-162`) below the
message body. Each answers a different question a reader cannot get from the
diff alone:

- **`Dacli-Agent: <id>`** — the exact agent that ran the command, e.g.
  `a-fixer-1p9jwx`. Lets a reader find every other thing that agent did in
  this run — its findings, decisions, and other commits (`dacli contrib`,
  `dacli blame`) — and, while its worktree still exists, its identity file
  under `.dacli/agents/<id>.md`.
- **`Dacli-Role: <role>`** — the role the agent was spawned under, e.g.
  `fixer`. Lets a reader roll defects up *by role* rather than by individual
  (agents are disposable; roles persist) — `dacli contrib`'s defect-rate
  column is keyed on this, and it is the signal for whether a role's prompt,
  scope, or model tier needs to change.
- **`Dacli-Task: <seq>-<slug>`** — the task the commit was claimed against,
  e.g. `354-document-agent-authored-commit-identity...`. Lets a reader open
  `.dacli/tasks/` (or the mirrored GitHub issue) and read the acceptance
  criteria, findings, and decisions that motivated the change — the *why*
  behind a diff that, on its own, only shows the *what*.

A commit missing `Dacli-Role` (root commits) or `Dacli-Task` (an untasked
commit) is normal; a commit missing `Dacli-Agent` entirely was not made
through `dacli commit` at all — see the next section.

`dacli blame` reads the trailers-plus-author back for reviewers; `dacli
contrib` rolls them up per role into a defect rate — which role produced which
class of finding, the signal for improving the agents.

## How agent-written code is reviewed before it lands

Two structurally different paths land a task's branch on trunk, and they carry
different amounts of review — this matters because which one runs is a config
choice (`dacli loop`'s `--pr` / `--no-pr`, or a per-spawn PR-first flag), not a
constant.

**PR-first (`landing.mode: pr`, `dacli integrate --pr`, or an explicit
`dacli loop --pr` — GITHUB.md § 9.5).** The branch is pushed, an enriched PR is
opened (acceptance criteria, findings, and — with `--with-verdicts` — the
verify-panel tally, all in the body), and by default it lands only once
`gh pr checks` reports every required check green — a real gate dacli cannot
bypass, but an automated one (CI, not a person). `--no-merge` is the mode
that actually stops for a human: the PR is left open and nothing merges until
someone does it by hand. Whether any given PR is read by a person before it
merges is not something git history can prove either way, but PR-first is
strictly more scrutinized than local integrate: it runs this repo's full CI
(build, vet, lint, race tests) that local integrate never invokes, and it
leaves the diff sitting on GitHub for as long as the operator lets it. This
repo's own history has a concrete example of that CI gate catching something a
local run missed: PR #429 (task 342) shipped a fix whose test assertions were
stale in a way the author's own `go test ./... | head` truncated past — CI's
untruncated run caught it before merge, and the fix landed as a same-PR
follow-up commit while the PR was still open, not a separate task filed after
the fact.

Configure the durable choice at creation with `dacli project add ...
--landing-mode pr --landing-base main`, or set `landing.mode` and
`landing.base` in the project frontmatter. Precedence is command-line override, project configuration, then
the legacy `local` default. `loop --dry-run` prints the effective mode, base,
override state, PR action, and gates. PR mode requires authenticated `gh`
(`gh auth login`; verify with `dacli github doctor`) and GitHub branch
protection configured with the intended required checks and review count.

The loop journals this resolved policy and the canonical task branch while a
push, PR, checks, or merge is in flight. Recover with `dacli loop status
--project <project>` and `dacli pr status --task <ref>`, repair authentication,
network, or CI, then rerun the bounded loop. It reuses the branch and PR. A PR
outage never falls back locally; `--no-pr` is an explicit audited override.

**Local integrate (`dacli integrate` / `dacli merge` / `dacli ship` with no
`--pr`).** This is a plain `git merge` (`mergeTask` in
`internal/features/vcs/lifecycle.go:1090`) run against each done task's
branch, one task at a time. **It reads no diff.** The only thing it inspects
is whether the merge produces a conflict — if it does, the task is blocked
and a finding is filed naming the conflicted files; if it does not, the
branch is merged, its worktree is torn down, and the branch is deleted. No
step in that path opens a file, runs a linter, or asks anything to judge
whether the change is *correct* — that question is left entirely to whatever
ran before the merge: the task's own test suite, its acceptance-criteria
checkboxes (`dacli task check`), and — only if the operator asked for it —
`dacli verify`'s adversarial panel, which argues about one **claim** in a
finding (majority vote across runtimes trying to refute it), not a
line-by-line review of the diff. `dacli loop`'s own periodic "review" phase
(`orchestration.go:1642`, `reviewPhase`) spawns an auditor role to find and
file *new* work for the backlog — it is a standing self-audit, not a gate on
the task that is about to merge.

Stated plainly: an unattended `dacli loop` run with `--no-pr` (or no `origin`
remote) can carry agent-written code from a claimed task all the way to
`main` with nobody, and nothing, ever having read the diff. Tests passing and
every acceptance box checked are real signal, but they are not code review,
and this doc should not let a reader assume otherwise.

## Parallel agents, isolated

`dacli next --parallel N` names the tasks worth running at once (zero-slack,
`SS`-safe). `dacli spawn --task X --worktree` then runs each child in its own
git worktree — a separate directory and branch over the shared object store —
so N agents work simultaneously without touching each other's files. Each
commits via `dacli commit` on its own branch, and the owner brings the work
back:

```
dacli spawn --task 001 --role builder --runtime cc --worktree   # parallel
dacli spawn --task 002 --role builder --runtime cc --worktree   # parallel
# ...each agent commits on its own branch, in its own worktree...
dacli integrate            # merge every done branch, in order
```

`dacli integrate` merges serialized, so a conflict surfaces one task at a time
rather than as a pile-up; a conflict **blocks that task and files a finding**
naming the conflicted files, and aborts the merge — dacli never leaves a
half-merged tree, because it cannot resolve conflicts and must not pretend to.
`dacli worktree add|list|remove`, `dacli push`, `dacli pr`, and
`dacli merge --task X` are the individual lifecycle steps.

### Where a worktree agent's writes go

A `--worktree` child sends its output to **two deliberately different places**,
and spawn tells the agent so up front (dacli 260):

- **Code** — file edits, `git`, and the commit `dacli commit` records — lands in
  the worktree, on that child's own branch. That is the isolation: N children
  never touch each other's tree.
- **Workspace state** — the agent's identity, `task check`, `note add`,
  findings, and the event crumb every `dacli commit` writes — resolves to the
  **shared main workspace** at the repo root, *not* the worktree's own `.dacli`.
  `workspace.Find` detects a linked worktree (via git's common dir) and redirects
  there.

This is intentional. A worktree checks out a git-tracked `.dacli` snapshot that
is stale the moment the branch was cut, so resolving state there would give the
child a *shadow* workspace: it could not see its own freshly-minted identity or
an uncommitted task, and its reports would never reach the owner. Sharing one
append-only store keeps concurrent writes safe and every record visible. The
consequence agents must know: **your code travels with your branch; your record
of the work lands in the shared store** — so never `cd` to the main checkout to
"fix" where a report went, which would only commit code onto `main`'s tree.

## Reporting problems with the tool

An agent that hits a bug in dacli *itself* (not its task) files it upstream
with `dacli report "<what dacli did wrong>"` — an explicit action, never
automatic, targeting this repo's issue tracker with version and environment
context. The self-improvement loop closes: bugs agents hit in the tool flow
back to the tool.

## Which commits are agent-authored, and how to tell

The trailer is the ground truth here, not the top-level git author line. As of
2026-08-11, `main` carries 616 commits:

```
$ git log main --oneline | wc -l
616
$ git log main --grep='^Dacli-Agent:' -E --oneline | wc -l
290
```

**290 of those 616 commits carry a `Dacli-Agent:` trailer** — proof they (or,
for a squashed PR, at least one commit folded into them) went through `dacli
commit` rather than a hand-run `git commit`. That is the number to trust for
"was this agent work", and it is why the count is taken by `--grep` over the
message rather than by the author line: this repo lands most PRs with a
GitHub **squash merge**, whose resulting commit is authored (git identity) by
whoever clicked merge on GitHub — a human, `Taha Bouhsine` — even when every
commit folded into it was authored by a dacli agent. GitHub's default squash
concatenates each constituent commit's full message, trailers included, into
the squash body, so `Dacli-Agent:` survives the squash even though the
author line does not. Counting `git log --format=%ae | grep @agent.dacli`
instead gives 221 — real, but an undercount, because it only sees commits
that were never squashed (direct `dacli integrate`/`dacli ship` merges and
`a-root`'s own record commits land straight on `main` without a PR).

**Not every merged PR is agent work, and the trailer says so honestly.** Some
PRs are a maintainer working through an interactive Claude Code session
rather than a spawned dacli agent — real work, real review, but never routed
through `dacli commit`, so those commits carry `Co-Authored-By: Claude Opus 5`
(the interactive-session convention) and no `Dacli-Agent:` trailer at all.
PR #440 (issue #437) is one: `git log -1 --format=%B a007429` shows five
`Co-Authored-By:` lines and zero `Dacli-Agent:` lines. Treat any claim that
"every commit here is agent-authored" the way this project treats an
unverified self-report about a task: a lead to check against the trailer, not
a fact to repeat.

**To identify one specific commit:** `git log -1 --format=%B <sha>` and look
for `Dacli-Agent:` / `Dacli-Role:` / `Dacli-Task:` in the body — never trust
`%an`/`%ae` alone, since a squash-merge author is the human who merged it, not
who wrote it.

## History note

The commits before task attribution existed were authored directly (`Taha
Bouhsine`), during the initial build-out when the attribution machinery did
not yet exist or was not yet dogfooded. From there on, dacli's own work is
authored by dacli agents — see the section above for exactly how much, and how
to check any single commit yourself.
