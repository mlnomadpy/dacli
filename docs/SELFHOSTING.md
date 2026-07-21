# dacli builds dacli

dacli is developed using dacli. Its own remaining work lives in `.dacli/`
(committed to this repo), picked in `dacli next` order, each task claimed,
verified against its acceptance criteria, and retro'd through the tool. One
feature was hardened by a real spawned opus reviewer; several bugs were caught
by dogfooding that the test suite had blessed.

## Commits are authored by dacli agents

Development commits are made through `dacli commit`, so `git log` and
`git blame` answer *which agent, in what role* wrote each line — the same
attribution any team using dacli gets:

```
$ git log --format='%an  <%ae>'
a-khwzk4bfr6 (maintainer)  <a-khwzk4bfr6@agent.dacli>
```

The flow, which is exactly what the `git_workflow` prompt tells every rw
agent:

```
git checkout -b dacli/<change>
DACLI_AGENT=<maintainer-token> dacli commit "<what changed>" --task <ref>
git checkout main && git merge --ff-only dacli/<change>   # attribution preserved
```

`dacli commit` refuses to commit on the default branch (the git-discipline
rule, enforced not just prompted), sets the git author to the agent and role,
and stamps `Dacli-Agent` / `Dacli-Role` / `Dacli-Task` trailers. `dacli blame`
reads it back for reviewers; `dacli contrib` rolls it up per role into a
defect rate — which role produced which class of finding, the signal for
improving the agents.

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
