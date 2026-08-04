# dacli builds dacli

dacli is developed using dacli. Its own remaining work lives in `.dacli/`
(committed to this repo), picked in `dacli next` order, each task claimed,
verified against its acceptance criteria, and retro'd through the tool. One
feature was hardened by a real spawned opus reviewer; several bugs were caught
by dogfooding that the test suite had blessed.

As of 2026-08-04, **96 pull requests (#39–#293) have merged** into `main` —
every one authored by a dacli agent, on its own `dacli/<seq>-<slug>` branch, and
landed through `dacli integrate` / `dacli merge`, never a hand-run `git merge`.
That count is not a claim to trust on faith: it comes straight from git history
(`git log main` — count the `Merge pull request #NNN` and squash `(#NNN)`
subjects), so a reader can regenerate it. The commits before #39 were authored
directly, during the initial build-out (see *History note*).

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
and stamps `Dacli-Agent` / `Dacli-Role` / `Dacli-Task` trailers. `dacli blame`
reads it back for reviewers; `dacli contrib` rolls it up per role into a
defect rate — which role produced which class of finding, the signal for
improving the agents.

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

## History note

The commits before this point were authored directly (`Taha Bouhsine`),
during the initial build-out when the attribution machinery did not yet
exist or was not yet dogfooded. From here, dacli's own work is authored by
dacli agents.
